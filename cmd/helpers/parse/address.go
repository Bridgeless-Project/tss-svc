package parse

import (
	"crypto/elliptic"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var parseAddressCmd = &cobra.Command{
	Use:   "address [x-cord] [y-cord]",
	Short: "Parse eth address from the given point",
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
		// Marshalled point contains constant 0x04 first byte, we do not have to include it
		hash := crypto.Keccak256(marshalled[1:])

		// The Ethereum address is the last 20 bytes of the hash.
		address := common.BytesToAddress(hash[12:]) // hash[12:32]
		fmt.Println("Ethereum address:", address.Hex())

		return nil
	},
}
