package migrate

import (
	"github.com/hyle-team/tss-svc/cmd/utils"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Downgrades the database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := utils.Config(cmd)
		return execute(cfg, migrate.Down)
	},
}
