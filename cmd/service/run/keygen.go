package run

import "github.com/spf13/cobra"

var keygenCmd = &cobra.Command{
	Use: "keygen",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
