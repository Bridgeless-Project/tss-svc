package cmd

import (
	"os"

	"github.com/hyle-team/tss-svc/cmd/helpers/generate"
	"github.com/hyle-team/tss-svc/cmd/service"

	"github.com/spf13/cobra"
)

func Execute() {
	root := &cobra.Command{
		Use:   "tss-svc",
		Short: "Threshold Signature Scheme Service",
	}

	root.AddCommand(service.Cmd, generate.Cmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
