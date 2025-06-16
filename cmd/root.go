package cmd

import (
	"os"

	"github.com/Bridgeless-Project/tss-svc/cmd/helpers"
	"github.com/Bridgeless-Project/tss-svc/cmd/service"

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
