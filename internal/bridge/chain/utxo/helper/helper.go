package helper

import (
	"crypto/ecdsa"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
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
