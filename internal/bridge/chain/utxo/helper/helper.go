package helper

import (
	btccfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	bchcfg "github.com/gcash/bchd/chaincfg"
	utxotypes "github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/types"
)

type UtxoHelper interface {
	ScriptSupported(script []byte) bool
	IsOpReturnScript(scriptRaw []byte) bool

	AddressValid(string) bool
	ExtractScriptAddresses(scriptRaw []byte) ([]string, error)
	PayToAddrScript(addr string) ([]byte, error)

	CalculateSignatureHash(scriptRaw []byte, tx *wire.MsgTx, idx int, amt int64) ([]byte, error)
	MockSignatureScript(scriptRaw []byte, tx *wire.MsgTx, idx int, amt int64) ([]byte, error)
}

func NewUtxoHelper(
	chainType utxotypes.Type,
	network utxotypes.Network,
) UtxoHelper {
	switch chainType {
	case utxotypes.TypeBtc:
		var params *btccfg.Params
		switch network {
		case utxotypes.NetworkMainnet:
			params = &btccfg.MainNetParams
		case utxotypes.NetworkTestnet3:
			params = &btccfg.TestNet3Params
			// TODO: add support for testnet4
		}

		return NewBtcHelper(params)
	case utxotypes.TypeBch:
		var params *bchcfg.Params
		switch network {
		case utxotypes.NetworkMainnet:
			params = &bchcfg.MainNetParams
		case utxotypes.NetworkTestnet3:
			params = &bchcfg.TestNet3Params
		case utxotypes.NetworkTestnet4:
			params = &bchcfg.TestNet4Params
		}

		return NewBchHelper(params)
	}

	panic("unsupported chain subtype")
}
