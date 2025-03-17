package reshare

import "github.com/spf13/cobra"

func init() {
	registerCommands(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "reshare",
	Short: "Command for service migration during key resharing",
}

func registerCommands(cmd *cobra.Command) {
	cmd.AddCommand(reshareBtcCmd)
}
