package parse

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/helper"
	utxotypes "github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	network   = string(utxotypes.DefaultNetwork)
	chainType = string(utxotypes.DefaultType)
)

func init() {
	registerParseAddressUtxoFlags(parseAddressUtxoCmd)
}

func registerParseAddressUtxoFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&network, "network", "mainnet", "Network type (mainnet/testnet3/testnet4)")
	cmd.Flags().StringVar(&chainType, "chain-type", "btc", "Chain type (btc/bch)")
}

var parseAddressUtxoCmd = &cobra.Command{
	Use:   "address-utxo [x-cord] [y-cord]",
	Short: "Parse utxo address from the given point",
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

		var net, chain = utxotypes.Network(network), utxotypes.Type(chainType)
		if err := net.Validate(); err != nil {
			return errors.Wrap(err, "invalid network type")
		}
		if err := chain.Validate(); err != nil {
			return errors.Wrap(err, "invalid chain type")
		}

		hlp := helper.NewUtxoHelper(chain, net)
		pubkey := &ecdsa.PublicKey{Curve: crypto.S256(), X: xCord, Y: yCord}

		fmt.Println("Utxo address:", hlp.P2pkhAddress(pubkey))

		return nil
	},
}
