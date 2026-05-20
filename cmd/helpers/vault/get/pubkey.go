package get

import (
	"encoding/hex"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
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
		share, protocol, err := storage.GetTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get TSS share from vault")
		}

		switch protocol {
		case tss.ProtocolID_ECDSA:
			pubKey := share.(keygen.LocalPartySaveData).ECDSAPub.ToECDSAPubKey()
			fmt.Println("X coordinate:", pubKey.X)
			fmt.Println("Y coordinate:", pubKey.Y)
		case tss.ProtocolID_FROST:
			pubKey, err := share.(frost.Config).PublicKey.MarshalBinary()
			if err != nil {
				return errors.Wrap(err, "failed to decode pub key")
			}
			fmt.Println("PubKey :", hex.EncodeToString(pubKey))
		}

		return nil
	},
}
