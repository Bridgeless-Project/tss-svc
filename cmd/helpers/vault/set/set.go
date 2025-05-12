package set

import "github.com/spf13/cobra"

func init() {
	registerSetCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "set",
	Short: "Command for setting sensitive data in Vault",
}

func registerSetCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		cosmosAccountCmd,
		tssShareCmd,
		tlsCertCmd,
	)
}
