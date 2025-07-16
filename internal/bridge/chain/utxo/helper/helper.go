package helper

import (
	"crypto/ecdsa"

	utxotypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	btccfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	bchcfg "github.com/gcash/bchd/chaincfg"
)

type UtxoHelper interface {
	ScriptSupported(script []byte) bool
	RetrieveOpReturnData(script []byte) (string, error)

	P2pkhAddress(pk *ecdsa.PublicKey) string
	AddressValid(string) bool
	ExtractScriptAddresses(scriptRaw []byte) ([]string, error)
	PayToAddrScript(addr string) ([]byte, error)

	NewUnsignedTransaction(
		unspent []btcjson.ListUnspentResult,
		feeRate btcutil.Amount,
		outputs []*wire.TxOut,
		changeAddr string,
	) (*txauthor.AuthoredTx, error)
	CalculateSignatureHash(scriptRaw []byte, tx *wire.MsgTx, idx int, amt int64) ([]byte, error)
	MockSignatureScript(scriptRaw []byte, tx *wire.MsgTx, idx int, amt int64) ([]byte, error)

	InjectSignatures(tx *wire.MsgTx, signatures []*common.SignatureData, pk *ecdsa.PublicKey) error
	TxHash(tx *wire.MsgTx) string
}

func NewUtxoHelper(
	chainType utxotypes.Chain,
	network utxotypes.Network,
) UtxoHelper {
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
		}

		return NewBtcHelper(params)
	case utxotypes.ChainBch:
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
