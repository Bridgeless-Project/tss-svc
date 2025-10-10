package run

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/api"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/repository"
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
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/acceptor"
	evmSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/evm"
	tonSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/ton"
	utxoSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/utxo"
	zanoSigning "github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing/zano"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
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

	wg := new(sync.WaitGroup)
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
	sub := subscriber.NewSubmitEventSubscriber(dtb, cfg.TendermintHttpClient(), logger.WithField("component", "core_event_subscriber"), connector)
	fetcher := deposit.NewFetcher(clientsRepo, connector)

	apiServer := api.NewServer(
		cfg.ApiGrpcListener(),
		cfg.ApiHttpListener(),
		dtb,
		logger.WithField("component", "api_server"),
		clientsRepo,
		fetcher,
		broadcast.NewBroadcaster(parties, logger.WithField("component", "broadcaster")),
		account.CosmosAddress(),
		connector,
	)
	p2pServer := p2p.NewServer(
		cfg.P2pGrpcListener(),
		sessionManager,
		parties,
		*cert,
		logger.WithField("component", "p2p_server"),
	)

	wg.Add(1)

	// p2p server spin-up
	eg.Go(func() error {
		defer wg.Done()

		status := p2p.PartyStatus_PS_SIGN
		if syncEnabled {
			status = p2p.PartyStatus_PS_SYNC
		}
		p2pServer.SetStatus(status)

		return errors.Wrap(p2pServer.Run(ctx), "error while running p2p server")
	})

	// API server spin-up
	wg.Add(2)
	eg.Go(func() error {
		defer wg.Done()
		return errors.Wrap(apiServer.RunHTTP(ctx), "error while running API HTTP gateway")
	})
	eg.Go(func() error {
		defer wg.Done()
		return errors.Wrap(apiServer.RunGRPC(ctx), "error while running API GRPC server")
	})

	// sessions spin-up
	var snc *p2p.Syncer
	if syncEnabled {
		snc, err = p2p.NewSyncer(parties, p2p.PartyStatus_PS_SIGN)
		if err != nil {
			return errors.Wrap(err, "failed to create syncer")
		}
	}

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

			sess := configureSigningSession(sessParams, parties, *account, share, dtb, fetcher, logger, client, connector)

			wg.Add(1)
			eg.Go(func() error {
				defer wg.Done()
				return errors.Wrap(sess.Run(ctx), "error while running signing session")
			})

			sessionManager.Add(sess)

			return nil
		})
	}

	// additional deposit acceptor session
	wg.Add(1)
	eg.Go(func() error {
		defer wg.Done()

		depositAcceptorSession := acceptor.NewDepositAcceptorSession(
			parties,
			fetcher,
			dtb,
			logger.WithField("component", "deposit_acceptor_session"),
		)
		sessionManager.Add(depositAcceptorSession)
		depositAcceptorSession.Run(ctx)

		return nil
	})

	// Core deposit subscriber spin-up
	// subscriber runs two goroutines now
	wg.Add(2)
	eg.Go(func() error {
		defer wg.Done()

		return errors.Wrap(sub.Run(ctx), "error while running core deposit subscriber")
	})

	if syncEnabled {
		eg.Go(func() error {
			sessionsWg.Wait()

			logger.Info("all signing sessions are ready, starting p2p server in sign mode")
			p2pServer.SetStatus(p2p.PartyStatus_PS_SIGN)

			return nil
		})
	}

	err = eg.Wait()
	wg.Wait()

	return err
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
) (sess p2p.RunnableTssSession) {
	switch client.Type() {
	case chain.TypeEVM:
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
	}

	return sess
}
