package service

import (
	"github.com/hyle-team/tss-svc/cmd/service/migrate"
	"github.com/hyle-team/tss-svc/cmd/service/run"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/spf13/cobra"
)

func init() {
	registerServiceCommands(Cmd)
	utils.RegisterConfigFlag(Cmd)
}

func registerServiceCommands(cmd *cobra.Command) {
	cmd.AddCommand(migrate.Cmd)
	cmd.AddCommand(run.Cmd)
	cmd.AddCommand(signCmd)
}

var Cmd = &cobra.Command{
	Use:   "service",
	Short: "Command for running service operations",
}
