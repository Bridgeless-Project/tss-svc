package parse

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var network string

func init() {
	registerParseAddressBtcFlags(parseAddressBtcCmd)
}

func registerParseAddressBtcFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&network, "network", "mainnet", "Network type (mainnet/testnet)")
}

var parseAddressBtcCmd = &cobra.Command{
	Use:   "address-btc [x-cord] [y-cord]",
	Short: "Parse btc address from the given point",
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

		var chainParams *chaincfg.Params
		switch network {
		case "mainnet":
			chainParams = &chaincfg.MainNetParams
		case "testnet":
			chainParams = &chaincfg.TestNet3Params
		default:
			return errors.New("invalid network type")
		}

		pubkey := &ecdsa.PublicKey{Curve: crypto.S256(), X: xCord, Y: yCord}
		compressed := crypto.CompressPubkey(pubkey)
		pubKeyHash := btcutil.Hash160(compressed)

		// Generate P2PKH address
		addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, chainParams)
		if err != nil {
			return errors.Wrap(err, "failed to create address")
		}

		fmt.Println("Bitcoin address:", addr.EncodeAddress())

		return nil
	},
}
