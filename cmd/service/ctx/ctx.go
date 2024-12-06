package ctx

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/config"
	"github.com/spf13/cobra"
)

const (
	cfgKey = iota
)

func WithConfig(cmd *cobra.Command, cfg config.Config) *cobra.Command {
	cmd.SetContext(context.WithValue(cmd.Context(), cfgKey, cfg))

	return cmd
}

func Config(cmd *cobra.Command) config.Config {
	return cmd.Context().Value(cfgKey).(config.Config)
}
