package get

import (
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var pubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Get the TSS public key from the vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		share, err := storage.GetTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get TSS share from vault")
		}

		pubKey := share.ECDSAPub.ToECDSAPubKey()
		fmt.Println("X coordinate:", pubKey.X)
		fmt.Println("Y coordinate:", pubKey.Y)

		return nil
	},
}
