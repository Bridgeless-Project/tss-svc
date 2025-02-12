package parse

import (
	"crypto/elliptic"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var parsePubkeyCmd = &cobra.Command{
	Use:   "pubkey [x-cord] [y-cord]",
	Short: "Parse pubkey from the given point",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		xCord, ok := new(big.Int).SetString(args[0], 10)
		if !ok {
			return errors.New("failed to parse x-cord")
		}

		yCord, ok := new(big.Int).SetString(args[1], 10)
		if !ok {
			return errors.New("failed to parse y-cord")
		}

		marshalled := elliptic.Marshal(tss.S256(), xCord, yCord)
		// Marshalled point contains constant 0x04 first byte, we have to remove it
		fmt.Println("Pubkey:", hexutil.Encode(marshalled[1:]))

		key, err := crypto.UnmarshalPubkey(marshalled)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal pubkey")
		}

		compressed := crypto.CompressPubkey(key)
		fmt.Println("Compressed:", hexutil.Encode(compressed))

		return nil
	},
}
