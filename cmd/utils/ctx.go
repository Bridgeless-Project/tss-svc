package utils

import (
	"context"

	"github.com/hyle-team/tss-svc/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/distributed_lab/kit/kv"
)

const (
	cfgKey     = iota
	configFlag = "config"
)

func WithConfig(cmd *cobra.Command, cfg config.Config) *cobra.Command {
	cmd.SetContext(context.WithValue(cmd.Context(), cfgKey, cfg))

	return cmd
}

func Config(cmd *cobra.Command) config.Config {
	return cmd.Context().Value(cfgKey).(config.Config)
}

func ConfigFromFlags(cmd *cobra.Command) (config.Config, error) {
	configPath, err := cmd.Flags().GetString(configFlag)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config flag")
	}

	// ensure that the viper is loaded
	viper := kv.NewViperFile(configPath)
	if _, err = viper.GetStringMap("ping"); err != nil {
		return nil, errors.Wrap(err, "failed to ping viper")
	}

	return config.New(viper), nil
}
