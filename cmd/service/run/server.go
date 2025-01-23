package run

import (
	"context"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/api"
	"github.com/hyle-team/tss-svc/internal/bridge/client"
	core "github.com/hyle-team/tss-svc/internal/core/connector"
	pg "github.com/hyle-team/tss-svc/internal/db/postgres"
	processor2 "github.com/hyle-team/tss-svc/internal/processor"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sync"
)

func init() {
	utils.RegisterOutputFlags(serverCmd)
}

var serverCmd = &cobra.Command{
	Use: "server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}
		logger := cfg.Log()
		// Configure chains map from config
		chains := cfg.Chains()
		clientsRepo, err := client.NewclientsRepository(chains, logger)
		if err != nil {
			return errors.Wrap(err, "failed to create clients repository")
		}
		db := pg.NewDepositsQ(cfg.DB())
		connector := core.NewConnector(cfg.CoreConnectorConfig().Connection, cfg.CoreConnectorConfig().Settings)
		processor := processor2.NewProcessor(clientsRepo, db, connector)
		srv := api.NewServer(
			cfg.GRPCListener(),
			cfg.HTTPListener(),
			db,
			logger.WithField("serviceComponent", "componentServer"),
			clientsRepo,
			processor,
		)
		wg := sync.WaitGroup{}
		wg.Add(2)

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
		wg.Wait()
		return nil
	},
}
