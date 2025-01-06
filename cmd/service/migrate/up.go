package migrate

import (
	"github.com/hyle-team/tss-svc/cmd/utils"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Upgrades the database with migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := utils.Config(cmd)
		return execute(cfg, migrate.Up)
	},
}
