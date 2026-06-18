package parse

import (
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var parseAddressEthCmd = &cobra.Command{
	Use:   "address-eth [x-cord] [y-cord]",
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

		fmt.Println("Ethereum address:", evm.PubkeyToAddress(xCord, yCord).Hex())

		return nil
	},
}
