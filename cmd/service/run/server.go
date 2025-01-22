package run

import (
	"context"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/api"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	pg "github.com/hyle-team/tss-svc/internal/db/postgres"
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
		chainsMap := make(map[string]chain.Chain)
		for _, chain := range chains {
			chainsMap[chain.Id] = chain
		}
		srv := api.NewServer(
			cfg.GRPCListener(),
			cfg.HTTPListener(),
			pg.NewDepositsQ(cfg.DB()),
			logger.WithField("serviceComponent", "componentServer"),
			chainsMap,
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
