package set

import (
	"os"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	tss2 "github.com/Bridgeless-Project/tss-svc/internal/tss"
	tss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var tssShareCmd = &cobra.Command{
	Use:  "tss-share [path-to-share-json] [protocol]",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sharePath := args[0]
		protocol := args[1]

		raw, err := os.ReadFile(sharePath)
		if err != nil {
			return errors.Wrap(err, "failed to read TSS share file")
		}

		var share tss2.Share
		switch protocol {
		case "ecdsa":
			share = tss.NewEcdsaShare()
			if err := share.Unmarshal(raw); err != nil {
				return errors.Wrap(err, "failed to unmarshal TSS share")
			}

		case "frost":
			share = tss.NewEcdsaShare()
			if err := share.Unmarshal(raw); err != nil {
				return errors.Wrap(err, "failed to unmarshal TSS share")
			}

		default:
			return errors.New("protocol not supported")
		}

		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()

		bytes, err := share.Marshal()
		if err != nil {
			return errors.Wrap(err, "failed to marshal TSS share")
		}

		if err := storage.SaveTssShare(secrets.TssShareKey(share.GetVaultPath()), bytes); err != nil {
			return errors.Wrap(err, "failed to save TSS share to vault")
		}
		config.Log().Info("TSS share was successfully saved")

		return nil
	},
}
