package run

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/api"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/repository"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/config"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	pg "github.com/Bridgeless-Project/tss-svc/internal/db/postgres"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Starts the service API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		err = runServiceApiMode(ctx, cfg)

		return errors.Wrap(err, "failed to run service api")
	},
}

func runServiceApiMode(ctx context.Context, cfg config.Config) error {
	logger := cfg.Log()
	storage := cfg.SecretsStorage()
	account, err := storage.GetCoreAccount()
	if err != nil {
		return errors.Wrap(err, "failed to get core account")
	}
	connector, err := coreConnector.NewConnector(
		*account,
		cfg.CoreConnectorConfig().Connection,
		cfg.CoreConnectorConfig().Settings,
		logger.WithField("component", "core_connector"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create core connector")
	}
	clientsRepo := repository.NewClientsRepository(cfg.Clients())
	fetcher := deposit.NewFetcher(clientsRepo, connector)
	dtb := pg.NewDepositsQ(cfg.DB())

	apiServer := api.NewServer(
		cfg.ApiGrpcListener(),
		cfg.ApiHttpListener(),
		dtb,
		logger.WithField("component", "api_server"),
		clientsRepo,
		fetcher,
		connector,
	)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return errors.Wrap(apiServer.RunHTTP(ctx), "error while running API HTTP gateway") })
	eg.Go(func() error { return errors.Wrap(apiServer.RunGRPC(ctx), "error while running API GRPC server") })

	return eg.Wait()
}
