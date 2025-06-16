package vault

import (
	"github.com/Bridgeless-Project/tss-svc/cmd/helpers/vault/get"
	"github.com/Bridgeless-Project/tss-svc/cmd/helpers/vault/set"
	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/spf13/cobra"
)

func init() {
	registerVaultCommands(Cmd)
	utils.RegisterConfigFlag(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "vault",
	Short: "Command for running Vault operations",
}

func registerVaultCommands(cmd *cobra.Command) {
	cmd.AddCommand(set.Cmd, get.Cmd)
}
