package set

import (
	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	registerCosmosAccountFlags(cosmosAccountCmd)
}

func registerCosmosAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&hrp, "hrp", "bridge", "Bech32 human-readable part of address")
}

var hrp string

var cosmosAccountCmd = &cobra.Command{
	Use:  "cosmos-account [priv-key]",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		privKey := args[0]

		acc, err := core.NewAccount(privKey, hrp)
		if err != nil {
			return errors.Wrap(err, "failed to parse Cosmos account")
		}

		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		if err = storage.SaveCoreAccount(acc); err != nil {
			return errors.Wrap(err, "failed to save account to vault")
		}

		config.Log().Info("Cosmos account was successfully saved")

		return nil
	},
}
