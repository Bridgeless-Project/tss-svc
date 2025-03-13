package run

import (
	"context"
	"fmt"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p/syncer"
	"gitlab.com/distributed_lab/logan/v3"
	"golang.org/x/sync/errgroup"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/api"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chains"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/bitcoin"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/repository"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/zano"
	"github.com/hyle-team/tss-svc/internal/config"
	coreConnector "github.com/hyle-team/tss-svc/internal/core/connector"
	"github.com/hyle-team/tss-svc/internal/core/subscriber"
	pg "github.com/hyle-team/tss-svc/internal/db/postgres"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var syncEnabled bool

func init() {
	registerSyncFlag(signCmd)
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

		if err = runSigningService(ctx, cfg); err != nil {
			cancel()
		}
		return errors.Wrap(err, "failed to run signing service")
	},
}

type RunnableTssSession interface {
	Run(context.Context) error
	p2p.TssSession
}

func runSigningService(ctx context.Context, cfg config.Config) error {
	wg := new(sync.WaitGroup)
	errGroup, ctx := errgroup.WithContext(ctx)
	logger := cfg.Log()
	chns := cfg.Chains()
	storage := vault.NewStorage(cfg.VaultClient())

	account, err := storage.GetCoreAccount()
	if err != nil {
		return errors.Wrap(err, "failed to get core account")
	}
	share, err := storage.GetTssShare()
	if err != nil {
		return errors.Wrap(err, "failed to get tss share")
	}
	clientsRepo, err := repository.NewClientsRepository(chns)
	if err != nil {
		return errors.Wrap(err, "failed to create clients repository")
	}
	sessionManager := p2p.NewSessionManager()
	dtb := pg.NewDepositsQ(cfg.DB())
	connector := coreConnector.NewConnector(*account, cfg.CoreConnectorConfig().Connection, cfg.CoreConnectorConfig().Settings)
	sub := subscriber.NewSubmitSubscriber(dtb, cfg.TendermintHttpClient(), logger.WithField("component", "core_event_subscriber"))
	fetcher := bridge.NewDepositFetcher(clientsRepo, connector)

	//
	//	once := sync.Once{}

	srv := api.NewServer(
		cfg.ApiGrpcListener(),
		cfg.ApiHttpListener(),
		dtb,
		logger.WithField("component", "server"),
		clientsRepo,
		fetcher,
		p2p.NewBroadcaster(cfg.Parties(), cfg.Log().WithField("component", "broadcaster")),
		account.CosmosAddress(),
		connector,
	)

	server := p2p.NewServer(cfg.P2pGrpcListener(), sessionManager, cfg.Log().WithField("component", "p2p_server"))

	wg.Add(1)

	// p2p server spin-up
	errGroup.Go(func() error {
		defer wg.Done()
		if syncEnabled {
			server.SetStatus(p2p.PartyStatus_PS_SYNC)
		} else {
			server.SetStatus(p2p.PartyStatus_PS_SIGN)
		}
		if err = server.Run(ctx); err != nil {
			return errors.Wrap(err, "failed to start p2p server")
		}

		return nil
	})

	// API servers spin-up
	wg.Add(2)
	errGroup.Go(func() error {
		defer wg.Done()
		if err = srv.RunHTTP(ctx); err != nil {
			return errors.Wrap(err, "failed to run HTTP gateway")
		}
		return nil
	})

	errGroup.Go(func() error {
		defer wg.Done()
		if err = srv.RunGRPC(ctx); err != nil {
			return errors.Wrap(err, "failed to run GRPC gateway")
		}
		return nil
	})

	snc := syncer.NewSyncer(cfg.MaxRetries(), ctx)
	// sessions spin-up
	for _, chain := range chns {
		wg.Add(1)
		errGroup.Go(func() error {
			defer wg.Done()

			client, _ := clientsRepo.Client(chain.Id)
			var sessParams session.SigningSessionParams

			if syncEnabled {
				cfg.Log().WithField("component", "syncer").Info(fmt.Sprintf("starting syncer for chain %s", chain.Id))
				connection, err := snc.FindPartyToSync(cfg.Parties())
				if err != nil {
					return errors.Wrap(err, "failed to find party to sync")
				}
				info, err := snc.WithParty(connection).Sync(chain.Id)
				if err != nil {
					return errors.Wrap(err, "failed to sync")
				}

				if info.Id == 0 {
					return errors.New("failed to get info, id is 0")
				}

				sessParams = session.SigningSessionParams{
					SessionParams: tss.SessionParams{
						Id:        info.Id,
						StartTime: time.Unix(info.NextSessionStartTime, 0),
						Threshold: cfg.TssSessionParams().Threshold,
					},
					ChainId: client.ChainId(),
				}
			} else {
				sessParams = session.SigningSessionParams{
					SessionParams: cfg.TssSessionParams(),
					ChainId:       client.ChainId(),
				}
			}

			sess := configureSession(sessParams, chain, cfg, account, share, dtb, fetcher, logger, client, connector)

			wg.Add(1)
			errGroup.Go(func() error {
				defer wg.Done()
				if err = sess.Run(ctx); err != nil {
					return errors.Wrap(err, "failed to run session")
				}
				server.SetStatus(p2p.PartyStatus_PS_SIGN)
				return nil
			})

			sessionManager.Add(sess)
			return nil
		})
	}

	// additional deposit acceptor session
	wg.Add(1)
	errGroup.Go(func() error {
		defer wg.Done()

		depositAcceptorSession := bridge.NewDepositAcceptorSession(
			cfg.Parties(),
			fetcher,
			dtb,
			logger.WithField("component", "deposit_acceptor_session"),
		)
		sessionManager.Add(depositAcceptorSession)
		depositAcceptorSession.Run(ctx)

		return nil
	})

	// Core deposit subscriber spin-up
	wg.Add(1)
	errGroup.Go(func() error {
		defer wg.Done()

		if err = sub.Run(ctx); err != nil {
			return errors.Wrap(err, "failed to run Core event subscriber")
		}
		return nil
	})

	err = errGroup.Wait()
	wg.Wait()

	return err
}

func configureSession(params session.SigningSessionParams,
	chain chains.Chain,
	cfg config.Config,
	account *core.Account,
	share *keygen.LocalPartySaveData,
	db db.DepositsQ,
	fetcher *bridge.DepositFetcher,
	logger *logan.Entry,
	client clients.Client,
	connector *coreConnector.Connector) (sess RunnableTssSession) {
	switch chain.Type {
	case chains.TypeEVM:
		evmSession := session.NewEvmSigningSession(
			tss.LocalSignParty{
				Address:   account.CosmosAddress(),
				Share:     share,
				Threshold: params.Threshold,
			},
			cfg.Parties(),
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*evm.Client)).WithCoreConnector(connector)
		sess = evmSession
	case chains.TypeZano:
		zanoSession := session.NewZanoSigningSession(
			tss.LocalSignParty{
				Address:   account.CosmosAddress(),
				Share:     share,
				Threshold: params.Threshold,
			},
			cfg.Parties(),
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*zano.Client)).WithCoreConnector(connector)
		sess = zanoSession
	case chains.TypeBitcoin:
		btcSession := session.NewBitcoinSigningSession(
			tss.LocalSignParty{
				Address:   account.CosmosAddress(),
				Share:     share,
				Threshold: params.Threshold,
			},
			cfg.Parties(),
			params,
			db,
			logger.WithField("component", "signing_session"),
		).WithDepositFetcher(fetcher).WithClient(client.(*bitcoin.Client)).WithCoreConnector(connector)
		sess = btcSession
	}

	return sess
}

func registerSyncFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&syncEnabled, "sync", "s", syncEnabled, "Sync enabled/disabled (disabled default)")
}
