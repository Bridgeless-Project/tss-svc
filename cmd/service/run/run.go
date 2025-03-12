package run

import (
	"github.com/hyle-team/tss-svc/cmd/service/run/reshare"
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
	cmd.AddCommand(keygenCmd, signCmd, reshare.Cmd)
}
