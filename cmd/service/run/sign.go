package run

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/core/subscriber"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	pg "github.com/Bridgeless-Project/tss-svc/internal/db/postgres"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/commissions"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/distributor"
	evmCentralized "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm/centralized"
	evmMerklized "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm/merklized"
	evmSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm/standart"
	solanaSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/solana"
	tonSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/ton"
	utxoSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/utxo"
	zanoSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/zano"
	"github.com/avast/retry-go"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
	share, err := storage.GetTssShare()
	if err != nil {
		return errors.Wrap(err, "failed to get tss share")
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
	commissionsSubscriber := subscriber.NewCommissionEventSubscriber(
		cfg.TendermintHttpClient(),
		connector,
		tss.LocalSignParty{
			Account:   *account,
			Share:     share,
			Threshold: cfg.TssSessionParams().Threshold,
		},
		parties,
		sessionManager,
		bridgeEvmSettings,
		logger.WithField("component", "commission_event_subscriber"),
	)

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

			sess := configureSigningSession(sessParams, parties, *account, share, dtb, fetcher, logger, client, connector, depositAcceptorSession)

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

	// Core commissions subscriber spin-up
	eg.Go(func() error {
		return errors.Wrap(commissionsSubscriber.Run(ctx), "error while running commissions subscriber")
	})

	go func() {
		withdrawals, err := commissions.NewSession(
			core.EventCommissionCollection{
				EventDataCommissionCollection: core.EventDataCommissionCollection{
					EpochId:     0,
					BlockHeight: 6388819,
				},
				Time: cfg.TssSessionParams().StartTime,
			},
			tss.LocalSignParty{
				Account:   *account,
				Share:     share,
				Threshold: cfg.TssSessionParams().Threshold,
			},
			parties,
			connector,
			bridgeEvmSettings,
			sessionManager,
			logger.WithField("component", "commissions_withdrawal_sessions"),
		).Run(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to run commissions_withdrawal_sessions")
			return
		} else if len(withdrawals) == 0 {
			logger.Warn("no commissions to withdraw")
			return
		}

		if err = retry.Do(func() error {
			return connector.SubmitSystemWithdrawals(ctx, withdrawals...)
		}, retry.Attempts(10), retry.Delay(3*time.Second)); err != nil {
			logger.WithError(err).Error("failed to submit commission withdrawals")
		}

		logger.WithField("count", len(withdrawals)).Info("successfully submitted commission withdrawals")
	}()

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
	account core.Account,
	share *keygen.LocalPartySaveData,
	db db.DepositsQ,
	fetcher *deposit.Fetcher,
	logger *logan.Entry,
	client chain.Client,
	connector *coreConnector.Connector,
	distributor *distributor.DepositDistributionSession,
) (sess p2p.RunnableTssSession) {
	switch client.Type() {
	case chain.TypeEVM:
		evmClient := client.(*evm.Client)
		switch {
		case evmClient.IsCentralized():
			sess = evmCentralized.NewSession(
				evmClient, db,
				logger.WithField("component", "centralized_signing_session"),
			)

		case evmClient.IsStandart():
			evmSession := evmSigning.NewSession(
				tss.LocalSignParty{
					Account:   account,
					Share:     share,
					Threshold: params.Threshold,
				},
				parties,
				params,
				db,
				logger.WithField("component", "signing_session"),
			).WithDepositFetcher(fetcher).WithClient(client.(*evm.Client)).WithCoreConnector(connector)
			if err := evmSession.Build(); err != nil {
				panic(errors.Wrap(err, "failed to build evm session"))
			}

			sess = evmSession
		default:
			evmMerklizedSession := evmMerklized.NewSession(
				tss.LocalSignParty{
					Account:   account,
					Share:     share,
					Threshold: params.Threshold,
				},
				parties,
				params,
				db,
				logger.WithField("component", "signing_session"),
			).WithDepositFetcher(fetcher).WithClient(client.(*evm.Client)).WithCoreConnector(connector).WithDistributor(distributor)
			if err := evmMerklizedSession.Build(); err != nil {
				panic(errors.Wrap(err, "failed to build evm session"))
			}
			sess = evmMerklizedSession
		}
	case chain.TypeZano:
		zanoSession := zanoSigning.NewSession(
			tss.LocalSignParty{
				Account:   account,
				Share:     share,
				Threshold: params.Threshold,
			},
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*zano.Client)).WithCoreConnector(connector)
		if err := zanoSession.Build(); err != nil {
			panic(errors.Wrap(err, "failed to build zano session"))
		}
		sess = zanoSession
	case chain.TypeBitcoin:
		btcSession := utxoSigning.NewSession(
			tss.LocalSignParty{
				Account:   account,
				Share:     share,
				Threshold: params.Threshold,
			},
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(utxoclient.Client)).WithCoreConnector(connector)
		if err := btcSession.Build(); err != nil {
			panic(errors.Wrap(err, "failed to build bitcoin session"))
		}
		sess = btcSession

	case chain.TypeTON:
		tonSession := tonSigning.NewSession(tss.LocalSignParty{
			Account:   account,
			Share:     share,
			Threshold: params.Threshold,
		},
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*ton.Client)).WithCoreConnector(connector)
		if err := tonSession.Build(); err != nil {
			panic(errors.Wrap(err, "failed to build TON session"))
		}
		sess = tonSession

	case chain.TypeSolana:
		solanaSession := solanaSigning.NewSession(
			tss.LocalSignParty{
				Account:   account,
				Share:     share,
				Threshold: params.Threshold,
			},
			parties,
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*solana.Client)).WithCoreConnector(connector)
		if err := solanaSession.Build(); err != nil {
			panic(errors.Wrap(err, "failed to build solana session"))
		}
		sess = solanaSession
	}

	return sess
}
