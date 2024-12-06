package migrate

import (
	servicectx "github.com/hyle-team/tss-svc/cmd/service/ctx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Downgrades the database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := servicectx.Config(cmd)
		return execute(cfg, migrate.Down)
	},
}
