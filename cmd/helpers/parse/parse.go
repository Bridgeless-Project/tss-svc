package parse

import "github.com/spf13/cobra"

func init() {
	registerParseCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "parse",
	Short: "Command for parsing data",
}

func registerParseCommands(cmd *cobra.Command) {
	cmd.AddCommand(parseAddressCmd, parsePubkeyCmd)
}
