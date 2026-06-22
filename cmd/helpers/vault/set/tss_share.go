package set

import (
	"os"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsa "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frost "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
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

		var share tss.Share
		switch protocol {
		case string(tss.ProtocolID_ECDSA):
			share = ecdsa.NewEcdsaShare()
			if err := share.Unmarshal(raw); err != nil {
				return errors.Wrap(err, "failed to unmarshal ECDSA TSS share")
			}

		case string(tss.ProtocolID_FROST):
			share = frost.NewFrostShare()
			if err := share.Unmarshal(raw); err != nil {
				return errors.Wrap(err, "failed to unmarshal FROST TSS share")
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
