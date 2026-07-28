package run

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/repository"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/solana"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/ton"
	utxoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/config"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/core/subscriber"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	pg "github.com/Bridgeless-Project/tss-svc/internal/db/postgres"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	frostTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/distributor"
	evmCentralized "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm/centralized"
	evmMerklized "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm/merklized"
	evmSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm/standart"
	solanaSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/solana"
	testSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/test"
	tonSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/ton"
	utxoSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/utxo"
	zanoSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/zano"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"gitlab.com/distributed_lab/logan/v3"
	"golang.org/x/sync/errgroup"
)

var syncEnabled bool

func init() {
	registerSyncFlag(signCmd)
}

func registerSyncFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&syncEnabled, "sync", "s", syncEnabled, "Sync mode enabled/disabled (disabled default)")
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Starts the service in the signing mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		err = runSigningServiceMode(ctx, cfg)

		return errors.Wrap(err, "failed to run signing service")
	},
}

func runSigningServiceMode(ctx context.Context, cfg config.Config) error {
	storage := cfg.SecretsStorage()
	account, err := storage.GetCoreAccount()
	bridgeEvmSettings := cfg.EvmSettings()
	if err != nil {
		return errors.Wrap(err, "failed to get core account")
	}

	cert, err := storage.GetLocalPartyTlsCertificate()
	if err != nil {
		return errors.Wrap(err, "failed to get local party tls certificate")
	}

	eg, ctx := errgroup.WithContext(ctx)
	logger := cfg.Log()
	clients := cfg.Clients()
	parties := cfg.Parties()
	clientsRepo := repository.NewClientsRepository(clients)
	sessionManager := p2p.NewSessionManager()
	dtb := pg.NewDepositsQ(cfg.DB())
	connector, err := coreConnector.NewConnector(
		*account,
		cfg.CoreConnectorConfig().Connection,
		cfg.CoreConnectorConfig().Settings,
		logger.WithField("component", "core_connector"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create core connector")
	}
	submitSubscriber := subscriber.NewSubmitEventSubscriber(
		dtb,
		cfg.TendermintHttpClient(),
		logger.WithField("component", "core_event_subscriber"),
		connector,
	)

	//TODO: add commission hanlder
	//commissionsSubscriber := subscriber.NewCommissionEventSubscriber(
	//	cfg.TendermintHttpClient(),
	//	connector,
	//	tss.LocalSignParty{
	//		Account:   *account,
	//		Share:     share,
	//		Threshold: cfg.TssSessionParams().Threshold,
	//	},
	//	parties,
	//	sessionManager,
	//	bridgeEvmSettings,
	//	logger.WithField("component", "commission_event_subscriber"),
	//)

	fetcher := deposit.NewFetcher(clientsRepo, connector, bridgeEvmSettings)

	p2pServer := p2p.NewServer(
		cfg.P2pGrpcListener(),
		sessionManager,
		parties,
		*cert,
		logger.WithField("component", "p2p_server"),
	)

	// p2p server spin-up
	eg.Go(func() error {
		status := p2p.PartyStatus_PS_SIGN
		if syncEnabled {
			status = p2p.PartyStatus_PS_SYNC
		}
		p2pServer.SetStatus(status)

		return errors.Wrap(p2pServer.Run(ctx), "error while running p2p server")
	})

	// sessions spin-up
	var snc *p2p.Syncer
	if syncEnabled {
		snc, err = p2p.NewSyncer(parties, p2p.PartyStatus_PS_SIGN)
		if err != nil {
			return errors.Wrap(err, "failed to create syncer")
		}
	}

	depositAcceptorSession := distributor.NewDepositDistributionSession(
		account.CosmosAddress(),
		parties,
		fetcher,
		dtb,
		logger.WithField("component", "deposit_distribution_session"),
	)

	sessionsWg := new(sync.WaitGroup)
	for _, client := range clients {
		sessionsWg.Add(1)
		eg.Go(func() error {
			defer sessionsWg.Done()

			var sessParams session.SigningParams

			if syncEnabled {
				logger.Infof("syncing next session params for chain %s", client.ChainId())
				sessionInfo, err := snc.Sync(ctx, client.ChainId())
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("failed to sync session info for chain %s", client.ChainId()))
				}
				sessParams = session.ParamsFromSigningSessionInfo(sessionInfo)
				logger.Infof("next session params for chain %s synced", client.ChainId())
			} else {
				sessParams = session.SigningParams{
					Params:  cfg.TssSessionParams(),
					ChainId: client.ChainId(),
				}
			}

			sess, err := configureSigningSession(
				sessParams,
				parties,
				tss.LocalSignParty{
					Account:   *account,
					Threshold: sessParams.Threshold,
					Share:     client.Share(),
				},
				dtb,
				fetcher,
				logger,
				client,
				connector,
				depositAcceptorSession,
			)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to configure signing session for chain %s", client.ChainId()))
			}

			eg.Go(func() error { return errors.Wrap(sess.Run(ctx), "error while running signing session") })

			sessionManager.Add(sess)

			return nil
		})
	}

	// additional deposit acceptor session
	eg.Go(func() error {
		sessionManager.Add(depositAcceptorSession)
		depositAcceptorSession.Run(ctx)

		return nil
	})

	// Core deposit subscriber spin-up
	eg.Go(func() error {
		return errors.Wrap(submitSubscriber.Run(ctx), "error while running core deposit subscriber")
	})

	//// Core commissions subscriber spin-up
	//eg.Go(func() error {
	//	return errors.Wrap(commissionsSubscriber.Run(ctx), "error while running commissions subscriber")
	//})

	if syncEnabled {
		eg.Go(func() error {
			sessionsWg.Wait()

			logger.Info("all signing sessions are ready, starting p2p server in sign mode")
			p2pServer.SetStatus(p2p.PartyStatus_PS_SIGN)

			return nil
		})
	}

	return eg.Wait()
}

func configureSigningSession(
	params session.SigningParams,
	parties []p2p.Party,
	localParty tss.LocalSignParty,
	db db.DepositsQ,
	fetcher *deposit.Fetcher,
	logger *logan.Entry,
	client chain.Client,
	connector *coreConnector.Connector,

	distributor *distributor.DepositDistributionSession,
) (p2p.RunnableTssSession, error) {
	if !client.IsCentralized() && localParty.Share == nil {
		return nil, errors.Errorf("TSS share is not configured for chain %s", client.ChainId())
	}
	if err := ensureSigningProtocolSupported(client, localParty.Share); err != nil {
		return nil, err
	}
	if err := validateLoadedSigningShare(localParty, parties); err != nil {
		return nil, errors.Wrap(err, "invalid local signing share")
	}

	switch client.Type() {
	case chain.TypeEVM:
		evmClient := client.(*evm.Client)
		switch {
		case evmClient.IsCentralized():
			return evmCentralized.NewSession(
				evmClient, db,
				logger.WithField("component", "centralized_signing_session"),
			), nil
		}
	}

	switch client.Type() {
	case chain.TypeEVM:
		evmClient := client.(*evm.Client)
		switch {
		case evmClient.IsCentralized():
			return evmCentralized.NewSession(
				evmClient, db,
				logger.WithField("component", "centralized_signing_session"),
			), nil
		case evmClient.IsStandart():
			evmSession := evmSigning.NewSession(
				localParty,
				parties,
				params,
				db,
				logger.WithField("component", "signing_session"),
			).WithDepositFetcher(fetcher).WithClient(client.(*evm.Client)).WithCoreConnector(connector)
			if err := evmSession.Build(); err != nil {
				return nil, errors.Wrap(err, "failed to build evm session")
			}

			return evmSession, nil
		default:
			evmMerklizedSession := evmMerklized.NewSession(
				localParty,
				parties,
				params,
				db,
				logger.WithField("component", "signing_session"),
			).WithDepositFetcher(fetcher).WithClient(client.(*evm.Client)).WithCoreConnector(connector).WithDistributor(distributor)
			if err := evmMerklizedSession.Build(); err != nil {
				return nil, errors.Wrap(err, "failed to build evm session")
			}

			return evmMerklizedSession, nil
		}
	case chain.TypeZano:
		zanoSession := zanoSigning.NewSession(
			localParty,
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*zano.Client)).WithCoreConnector(connector)
		if err := zanoSession.Build(); err != nil {
			return nil, errors.Wrap(err, "failed to build zano session")
		}

		return zanoSession, nil
	case chain.TypeBitcoin:
		btcSession := utxoSigning.NewSession(
			localParty,
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(utxoclient.Client)).WithCoreConnector(connector)
		if err := btcSession.Build(); err != nil {
			return nil, errors.Wrap(err, "failed to build bitcoin session")
		}

		return btcSession, nil
	case chain.TypeTON:
		tonSession := tonSigning.NewSession(localParty,
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*ton.Client)).WithCoreConnector(connector)
		if err := tonSession.Build(); err != nil {
			return nil, errors.Wrap(err, "failed to build TON session")
		}

		return tonSession, nil
	case chain.TypeSolana:
		solanaSession := solanaSigning.NewSession(
			localParty,
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*solana.Client)).WithCoreConnector(connector)
		if err := solanaSession.Build(); err != nil {
			return nil, errors.Wrap(err, "failed to build solana session")
		}

		return solanaSession, nil
	case chain.TypeOther:
		testSession := testSigning.NewSession(
			localParty,
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"))
		if err := testSession.Build(); err != nil {
			return nil, errors.Wrap(err, "failed to build test session")
		}

		return testSession, nil
	default:
		return nil, errors.Errorf("unsupported chain type: %s", client.Type())
	}
}

func ensureSigningProtocolSupported(client chain.Client, share tss.Share) error {
	if client.IsCentralized() {
		return nil
	}

	if share == nil {
		return errors.Errorf("TSS share is not configured for chain %s", client.ChainId())
	}
	if err := chain.ValidateSigningProtocol(client.Type(), share.Protocol()); err != nil {
		return errors.Wrapf(err, "chain %s does not support configured signing protocol", client.ChainId())
	}
	return nil
}

func validateLoadedSigningShare(localParty tss.LocalSignParty, parties []p2p.Party) error {
	if localParty.Share == nil || localParty.Share.Protocol() != tss.ProtocolID_FROST {
		return nil
	}
	frostShare, ok := localParty.Share.(*frostTss.FrostShare)
	if !ok {
		return errors.New("FROST share has an unexpected implementation")
	}
	participantIDs := make([]party.ID, 0, len(parties)+1)
	participantIDs = append(participantIDs, party.ID(localParty.Account.CosmosAddress().String()))
	for _, remote := range parties {
		participantIDs = append(participantIDs, party.ID(remote.CoreAddress.String()))
	}
	return frostShare.Validate(
		party.ID(localParty.Account.CosmosAddress().String()),
		localParty.Threshold,
		participantIDs,
	)
}
