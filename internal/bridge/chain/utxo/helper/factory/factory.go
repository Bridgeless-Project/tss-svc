package factory

import (
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/bch"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/btc"
	utxotypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	btccfg "github.com/btcsuite/btcd/chaincfg"
	bchcfg "github.com/gcash/bchd/chaincfg"
)

func NewUtxoHelper(
	chainType utxotypes.Chain,
	network utxotypes.Network,
) helper.UtxoHelper {
	switch chainType {
	case utxotypes.ChainBtc:
		var params *btccfg.Params
		switch network {
		case utxotypes.NetworkMainnet:
			params = &btccfg.MainNetParams
		case utxotypes.NetworkTestnet3:
			params = &btccfg.TestNet3Params
		case utxotypes.NetworkTestnet4:
			// TODO: add support for testnet4
			panic("testnet4 is not yet supported for BTC")
		default:
			panic(fmt.Sprintf("unknown network: %s", network))
		}

		return btc.NewHelper(params)
	case utxotypes.ChainBch:
		var params *bchcfg.Params
		switch network {
		case utxotypes.NetworkMainnet:
			params = &bchcfg.MainNetParams
		case utxotypes.NetworkTestnet3:
			params = &bchcfg.TestNet3Params
		case utxotypes.NetworkTestnet4:
			params = &bchcfg.TestNet4Params
		default:
			panic(fmt.Sprintf("unknown network: %s", network))
		}

		return bch.NewHelper(params)
	}

	panic(fmt.Sprintf("unknown chain type: %s", chainType))
}
