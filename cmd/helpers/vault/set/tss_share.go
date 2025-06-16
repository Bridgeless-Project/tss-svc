package set

import (
	"encoding/json"
	"os"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var tssShareCmd = &cobra.Command{
	Use:  "tss-share [path-to-share-json]",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sharePath := args[0]

		raw, err := os.ReadFile(sharePath)
		if err != nil {
			return errors.Wrap(err, "failed to read TSS share file")
		}

		var share *keygen.LocalPartySaveData
		if err := json.Unmarshal(raw, &share); err != nil {
			return errors.Wrap(err, "failed to unmarshal TSS share")
		}

		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		if err := storage.SaveTssShare(share); err != nil {
			return errors.Wrap(err, "failed to save TSS share to vault")
		}

		config.Log().Info("TSS share was successfully saved")

		return nil
	},
}
