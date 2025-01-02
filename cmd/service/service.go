package service

import (
	servicectx "github.com/hyle-team/tss-svc/cmd/service/ctx"
	"github.com/hyle-team/tss-svc/cmd/service/migrate"
	"github.com/hyle-team/tss-svc/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/distributed_lab/kit/kv"
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
		configPath, _ := cmd.Flags().GetString(configFlag)
		viper := kv.NewViperFile(configPath)

		// ensure that the viper is loaded
		if _, err := viper.GetStringMap("ping"); err != nil {
			return errors.Wrap(err, "failed to ping viper")
		}

		cfg := config.New(viper)
		cmd = servicectx.WithConfig(cmd, cfg)

		return nil
	},
}
