package run

import (
	"github.com/spf13/cobra"
)

func init() {
	registerCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Command for running service",
}

func registerCommands(cmd *cobra.Command) {
	cmd.AddCommand(keygenCmd)
	cmd.AddCommand(signCmd)
}
