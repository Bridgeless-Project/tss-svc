package get

import (
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	ecdsaTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frostTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var pubkeyCmd = &cobra.Command{
	Use:   "pubkey [protocol]",
	Args:  cobra.ExactArgs(1),
	Short: "Get the TSS public key from the vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()

		// TODO: use consts
		switch args[0] {
		case "ecdsa":
			share := ecdsaTss.NewEcdsaShare()
			err := storage.LoadTssShare(share)
			if err != nil {
				return errors.Wrap(err, "failed to get TSS share from vault")
			}

			fmt.Println("pubkey:", share.PubKey())
		case "frost":
			share := frostTss.NewFrostShare()
			err := storage.LoadTssShare(share)
			if err != nil {
				return errors.Wrap(err, "failed to get TSS share from vault")
			}

			fmt.Println("PubKey :", share.PubKey())
		default:
			return errors.Errorf("unsupported TSS protocol: %s", args[0])
		}

		return nil
	},
}
