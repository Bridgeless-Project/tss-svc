package run

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/api"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chains"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/repository"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/zano"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	"github.com/hyle-team/tss-svc/internal/config"
	core "github.com/hyle-team/tss-svc/internal/core/connector"
	pg "github.com/hyle-team/tss-svc/internal/db/postgres"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

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

		wg := &sync.WaitGroup{}
		if err := runSigningService(ctx, cfg, wg); err != nil {
			return errors.Wrap(err, "failed to run signing service")
		}
		wg.Wait()

		return nil
	},
}

func runSigningService(ctx context.Context, cfg config.Config, wg *sync.WaitGroup) error {
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

	db := pg.NewDepositsQ(cfg.DB())
	connector := core.NewConnector(cfg.CoreConnectorConfig().Connection, cfg.CoreConnectorConfig().Settings)
	fetcher := bridge.NewDepositFetcher(clientsRepo, connector)
	srv := api.NewServer(
		cfg.ApiGrpcListener(),
		cfg.ApiHttpListener(),
		db,
		logger.WithField("component", "server"),
		clientsRepo,
		fetcher,
		p2p.NewBroadcaster(cfg.Parties()),
		account.CosmosAddress(),
	)

	// API servers spin-up
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := srv.RunHTTP(ctx); err != nil {
			logger.WithError(err).Error("rest gateway error occurred")
		}
	}()
	go func() {
		defer wg.Done()
		if err := srv.RunGRPC(ctx); err != nil {
			logger.WithError(err).Error("grpc server error occurred")
		}
	}()

	sessionManager := p2p.NewSessionManager()
	type RunnableTssSession interface {
		Run(context.Context) error
		p2p.TssSession
	}

	// sessions spin-up
	for _, chain := range chns {
		client, _ := clientsRepo.Client(chain.Id)
		sessParams := cfg.TSSParams().SigningSessionParams()

		var sess RunnableTssSession
		switch chain.Type {
		case chains.TypeEVM:
			constructor := withdrawal.NewEvmConstructor(client.(*evm.Client))
			evmSession := session.NewEvmSigningSession(
				tss.LocalSignParty{
					Address:   account.CosmosAddress(),
					Share:     share,
					Threshold: sessParams.Threshold,
				},
				cfg.Parties(),
				sessParams.WithChainId(client.ChainId()),
				db,
				logger.WithField("component", "signing_session"),
			).WithDepositFetcher(fetcher).WithConstructor(constructor)
			sess = evmSession
		case chains.TypeZano:
			constructor := withdrawal.NewZanoConstructor(client.(*zano.Client))
			zanoSession := session.NewZanoSigningSession(
				tss.LocalSignParty{
					Address:   account.CosmosAddress(),
					Share:     share,
					Threshold: sessParams.Threshold,
				},
				cfg.Parties(),
				sessParams.WithChainId(client.ChainId()),
				db,
				logger.WithField("component", "signing_session"),
			).WithDepositFetcher(fetcher).WithConstructor(constructor)
			sess = zanoSession
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := sess.Run(ctx); err != nil {
				logger.WithError(err).Error("failed to session")
			}
		}()

		sessionManager.Add(sess)
	}

	// additional deposit acceptor session
	wg.Add(1)
	go func() {
		defer wg.Done()

		depositAcceptorSession := bridge.NewDepositAcceptorSession(
			cfg.Parties(),
			fetcher,
			clientsRepo,
			db,
			logger.WithField("component", "deposit_acceptor_session"),
		)
		sessionManager.Add(depositAcceptorSession)
		depositAcceptorSession.Run(ctx)
	}()

	// p2p server spin-up
	wg.Add(1)
	go func() {
		defer wg.Done()

		server := p2p.NewServer(cfg.P2pGrpcListener(), sessionManager)
		server.SetStatus(p2p.PartyStatus_PS_SIGN)
		if err := server.Run(ctx); err != nil {
			logger.WithError(err).Error("failed to run p2p server")
		}
	}()

	return nil
}
