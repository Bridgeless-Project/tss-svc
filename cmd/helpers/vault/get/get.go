package get

import "github.com/spf13/cobra"

func init() {
	registerSetCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "get",
	Short: "Command for getting sensitive data from Vault",
}

func registerSetCommands(cmd *cobra.Command) {
	cmd.AddCommand(pubkeyCmd)
}
