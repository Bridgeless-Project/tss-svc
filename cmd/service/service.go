package service

import (
	"github.com/hyle-team/tss-svc/cmd/service/migrate"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	configFlag = "config"
)

func init() {
	registerServiceCommands(Cmd)
	registerServiceFlags(Cmd)
}

func registerServiceCommands(cmd *cobra.Command) {
	cmd.AddCommand(migrate.Cmd)
}

func registerServiceFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(configFlag, "config.yaml", "config file path")
}

var Cmd = &cobra.Command{
	Use:   "service",
	Short: "Command for running service operations",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		cmd = utils.WithConfig(cmd, cfg)

		return nil
	},
}
