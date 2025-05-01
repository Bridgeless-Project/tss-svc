package helpers

import (
	"github.com/hyle-team/tss-svc/cmd/helpers/generate"
	"github.com/hyle-team/tss-svc/cmd/helpers/parse"
	"github.com/hyle-team/tss-svc/cmd/helpers/vault"
	"github.com/spf13/cobra"
)

func init() {
	registerHelpersCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "helpers",
	Short: "Command for running helper operations",
}

func registerHelpersCommands(cmd *cobra.Command) {
	cmd.AddCommand(generate.Cmd, parse.Cmd, vault.Cmd)
}
