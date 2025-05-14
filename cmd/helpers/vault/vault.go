package vault

import (
	"github.com/hyle-team/tss-svc/cmd/helpers/vault/get"
	"github.com/hyle-team/tss-svc/cmd/helpers/vault/set"
	"github.com/hyle-team/tss-svc/cmd/utils"
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
