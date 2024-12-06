package migrate

import (
	"github.com/hyle-team/tss-svc/internal/assets"
	"github.com/hyle-team/tss-svc/internal/config"
	"github.com/pkg/errors"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

func init() {
	registerCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "migrate",
	Short: "Command for running database migrations",
}

func registerCommands(cmd *cobra.Command) {
	cmd.AddCommand(upCmd)
	cmd.AddCommand(downCmd)
}

func execute(cfg config.Config, direction migrate.MigrationDirection) error {
	migrationsFs := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: assets.Migrations,
		Root:       "migrations",
	}

	applied, err := migrate.Exec(cfg.DB().RawDB(), "postgres", migrationsFs, direction)
	if err != nil {
		return errors.Wrap(err, "failed to apply migrations")
	}

	cfg.Log().WithField("applied", applied).Info("migrations applied")

	return nil
}
