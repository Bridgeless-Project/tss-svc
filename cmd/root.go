package cmd

import (
	"github.com/hyle-team/tss-svc/cmd/helpers"
	"github.com/hyle-team/tss-svc/cmd/service"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	root := &cobra.Command{
		Use:   "tss-svc",
		Short: "Threshold Signature Scheme Service",
	}

	root.AddCommand(service.Cmd, helpers.Cmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
