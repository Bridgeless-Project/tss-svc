package run

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/core/subscriber"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/api"
	chainTypes "github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/client"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
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
	Use: "sign",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		logger := cfg.Log()
		chains := cfg.Chains()
		clientsRepo, err := client.NewRepository(chains)
		if err != nil {
			return errors.Wrap(err, "failed to create clients repository")
		}
		db := pg.NewDepositsQ(cfg.DB())
		connector := core.NewConnector(cfg.CoreConnectorConfig().Connection, cfg.CoreConnectorConfig().Settings)
		sub := subscriber.NewSubmitSubscriber(db, cfg.Client(), logger)
		pr := withdrawal.NewProcessor(clientsRepo, connector)
		srv := api.NewServer(
			cfg.ApiGrpcListener(),
			cfg.ApiHttpListener(),
			db,
			logger.WithField("component", "server"),
			clientsRepo,
			pr,
		)
		storage := vault.NewStorage(cfg.VaultClient())
		account, err := storage.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}
		share, err := storage.GetTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get tss share")
		}

		wg := sync.WaitGroup{}
		wg.Add(3)

		go func() {
			defer wg.Done()
			sub.Run(ctx)
		}()
		go func() {
			defer wg.Done()
			if err := srv.RunHTTP(context.Background()); err != nil {
				logger.WithError(err).Error("rest gateway error occurred")
			}
		}()

		go func() {
			defer wg.Done()
			if err := srv.RunGRPC(context.Background()); err != nil {
				logger.WithError(err).Error("grpc server error occurred")
			}
		}()

		sessionManager := p2p.NewSessionManager()
		for _, chain := range chains {
			if chain.Type != chainTypes.TypeEVM {
				continue
			}

			client, _ := clientsRepo.Client(chain.Id)
			sessParams := cfg.TSSParams().SigningSessionParams()
			constructor := withdrawal.NewEvmConstructor(client.(evm.BridgeClient))
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
			).WithProcessor(pr).WithConstructor(constructor)

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := evmSession.Run(ctx); err != nil {
					logger.WithError(err).Error("failed to run evm session")
				}
			}()

			sessionManager.Add(evmSession)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			server := p2p.NewServer(cfg.P2pGrpcListener(), sessionManager)
			server.SetStatus(p2p.PartyStatus_PS_KEYGEN)
			if err := server.Run(ctx); err != nil {
				logger.WithError(err).Error("failed to run p2p server")
			}
		}()

		wg.Wait()

		return nil
	},
}
