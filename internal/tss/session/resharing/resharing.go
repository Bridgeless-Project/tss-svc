package resharing

import (
	"context"
	"fmt"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	solanaclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/solana"
	utxoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	zanoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/solana"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/ton"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/utxo"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/zano"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"golang.org/x/sync/errgroup"
)

const InitializationGapDuration = time.Minute

type Session struct {
	params Params

	oldParties []p2p.Party
	oldParams  session.Params

	selfOld, selfNew bool

	secrets        secrets.Storage
	sessionManager *p2p.SessionManager
	core           *coreConnector.Connector
	chains         chain.Repository

	logger *logan.Entry
}

func NewSession(
	params Params,
	oldSet []p2p.Party,
	oldParams session.Params,
	secrets secrets.Storage,
	sessionManager *p2p.SessionManager,
	core *coreConnector.Connector,
	chains chain.Repository,
	logger *logan.Entry,
) *Session {
	selfOld, selfNew := true, true
	if params.NewParticipant {
		selfOld = false
	}
	if len(params.Parties) == 0 {
		selfNew = false
	}

	return &Session{
		params:         params,
		oldParties:     oldSet,
		secrets:        secrets,
		sessionManager: sessionManager,
		logger:         logger,
		core:           core,
		chains:         chains,
		oldParams:      oldParams,
		selfOld:        selfOld,
		selfNew:        selfNew,
	}
}

func (s *Session) Run(ctx context.Context) error {
	account, err := s.secrets.GetCoreAccount()
	if err != nil {
		return errors.Wrap(err, "failed to get core account")
	}
	state := resharingTypes.InitializeState(s.params.Epoch, s.params.Threshold, s.params.StartTime, account)

	keygenRound := NewKeygenHandler(s.params.Parties, s.secrets, s.core, s.sessionManager, s.logger, s.selfOld, s.selfNew)
	keygenManager := resharingTypes.NewHandlerManager(
		keygenRound, state, s.logger.WithField("component", "resharing_keygen_manager"),
	)

	if err = keygenManager.Manage(ctx, s.params.StartTime); err != nil {
		return errors.Wrap(err, "failed to manage keygen round")
	}

	sessionStartTime := s.params.StartTime.Add(keygenRound.MaxHandleDuration() + InitializationGapDuration)
	state.SessionStartTime = sessionStartTime

	// new party does not participate in migration
	if s.selfNew && !s.selfOld {
		return errors.Wrap(s.manageWallets(state), "failed to manage wallets for new party")
	}

	if err = s.runMigration(ctx, state); err != nil {
		return errors.Wrap(err, "failed to run state migration")
	}

	// old party won't participate further
	if s.selfOld && !s.selfNew {
		return nil
	}

	if err = s.manageWallets(state); err != nil {
		return errors.Wrap(err, "failed to manage wallets session")
	}

	if err = s.manageShares(state); err != nil {
		return errors.Wrap(err, "failed to manage key shares")
	}

	return nil
}

func (s *Session) runMigration(ctx context.Context, state *resharingTypes.State) error {
	share, err := s.secrets.GetTssShare()
	if err != nil {
		return errors.Wrap(err, "failed to get TSS share")
	}
	state.OldShare = share

	self := tss.LocalSignParty{
		Account:   *state.Account,
		Share:     share,
		Threshold: s.oldParams.Threshold,
	}

	var (
		managers              []*resharingTypes.HandlerManager
		evmSessionInitialized = false
	)

	for _, ch := range s.chains.Clients() {
		switch ch.Type() {
		case chain.TypeEVM:
			if evmSessionInitialized {
				continue
			}
			evmSessionInitialized = true
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					NewUpdateContractHandler(
						self,
						s.oldParties,
						s.sessionManager,
						bridgeTypes.ChainType_EVM,
						evm.NewAddSignerOperation(state.NewPubKey, state.GlobalStartTime),
						evm.NewRemoveSignerOperation(self.Share.ECDSAPub.ToECDSAPubKey(), state.GlobalStartTime),
						s.logger),
					state,
					s.logger.WithField("component", "evm_migration_manager"),
				),
			)
		case chain.TypeSolana:
			client := ch.(*solanaclient.Client)
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					NewUpdateContractHandler(
						self,
						s.oldParties,
						s.sessionManager,
						bridgeTypes.ChainType_SOLANA,
						solana.NewAddSignerOperation(state.NewPubKey, state.GlobalStartTime, client.BridgeId()),
						solana.NewRemoveSignerOperation(self.Share.ECDSAPub.ToECDSAPubKey(), state.GlobalStartTime, client.BridgeId()),
						s.logger),
					state,
					s.logger.WithField("component", fmt.Sprintf("solana_migration_manager_%s", ch.ChainId())),
				),
			)
		case chain.TypeTON:
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					NewUpdateContractHandler(
						self,
						s.oldParties,
						s.sessionManager,
						bridgeTypes.ChainType_TON,
						ton.NewAddSignerOperation(state.NewPubKey, state.GlobalStartTime),
						ton.NewRemoveSignerOperation(self.Share.ECDSAPub.ToECDSAPubKey(), state.GlobalStartTime),
						s.logger),
					state,
					s.logger.WithField("component", "ton_migration_manager"),
				),
			)
		case chain.TypeBitcoin:
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					utxo.NewHandler(self, s.oldParties, ch.(utxoclient.Client), s.sessionManager, s.logger),
					state,
					s.logger.WithField("component", fmt.Sprintf("utxo_migration_manager_%s", ch.ChainId())),
				),
			)
		case chain.TypeZano:
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					zano.NewHandler(self, s.oldParties, ch.(*zanoclient.Client), s.sessionManager, s.logger, s.core),
					state,
					s.logger.WithField("component", fmt.Sprintf("zano_migration_manager_%s", ch.ChainId())),
				),
			)
		default:
			continue
		}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, manager := range managers {
		eg.Go(func() error {
			return manager.Manage(egCtx, state.SessionStartTime)
		})
	}

	if err = eg.Wait(); err != nil {
		return errors.Wrap(err, "failed to run resharing session handlers")
	}

	submitManager := resharingTypes.NewHandlerManager(
		NewSubmitHandler(s.core, s.logger),
		state, s.logger.WithField("component", "resharing_submit_manager"),
	)
	if err = submitManager.Manage(ctx, time.Now()); err != nil {
		return errors.Wrap(err, "failed to manage submit handler")
	}

	return nil
}

func (s *Session) manageWallets(state *resharingTypes.State) error {
	s.logger.Info("updating chain wallets for new TSS share")

	eg := errgroup.Group{}
	for _, ch := range s.chains.Clients() {
		if ch.Type() != chain.TypeBitcoin {
			continue
		}

		eg.Go(func() error {
			utxoClient := ch.(utxoclient.Client)
			if err := utxoClient.InitializeWallet(state.NewPubKey, state.Epoch, state.GlobalStartTime); err != nil {
				return errors.Wrap(err, "failed to initialize wallet")
			}
			s.logger.Infof("successfully initialized wallet for chain %s", ch.ChainId())

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "failed to initialize wallets for new TSS share")
	}
	s.logger.Info("successfully initialized wallets for new TSS share")

	return nil
}

func (s *Session) manageShares(state *resharingTypes.State) error {
	if err := s.secrets.SaveTssShare(state.NewShare); err != nil {
		return errors.Wrap(err, "failed to save new TSS share")
	}
	if err := s.secrets.SaveTemporaryTssShare(state.OldShare); err != nil {
		return errors.Wrap(err, "failed to save old TSS share")
	}

	return nil
}
