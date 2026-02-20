package resharing

import (
	"context"
	"fmt"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	utxoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	zanoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/evm"
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
	state := resharingTypes.InitializeState(s.params.Epoch, s.params.StartTime, account)

	keygenRound := NewKeygenHandler()
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
					evm.NewHandler(self, s.oldParties, s.sessionManager, s.logger),
					state,
					s.logger.WithField("component", "resharing_evm_migration_manager"),
				),
			)
		case chain.TypeBitcoin:
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					utxo.NewHandler(self, s.oldParties, ch.(utxoclient.Client), s.sessionManager, s.logger),
					state,
					s.logger.WithField("component", fmt.Sprintf("resharing_utxo_migration_manager_%s", ch.ChainId())),
				),
			)
		case chain.TypeZano:
			managers = append(managers,
				resharingTypes.NewHandlerManager(
					zano.NewHandler(self, s.oldParties, ch.(*zanoclient.Client), s.sessionManager, s.logger, s.core),
					state,
					s.logger.WithField("component", fmt.Sprintf("resharing_zano_migration_manager_%s", ch.ChainId())),
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
	// TODO: implement me
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
