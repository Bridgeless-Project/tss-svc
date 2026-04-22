package parse

import (
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
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

		fmt.Println("Pubkey:", bridge.PubkeyToString(xCord, yCord))
		fmt.Println("Pubkey [compressed]:", bridge.PubkeyCompressedToString(xCord, yCord))

		return nil
	},
}
