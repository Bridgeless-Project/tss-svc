package generate

import "github.com/spf13/cobra"

func init() {
	registerGenerateCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "generate",
	Short: "Command for generating data",
}

func registerGenerateCommands(cmd *cobra.Command) {
	cmd.AddCommand(preparamsCmd)
}
