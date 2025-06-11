package utxo

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/btcsuite/btcd/wire"
)

func ToAmount(val float64) *big.Int {
	bigval := new(big.Float).SetFloat64(val)

	coin := new(big.Float)
	coin.SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil))

	bigval.Mul(bigval, coin)

	result := new(big.Int)
	bigval.Int(result)

	return result
}

func EncodeTransaction(tx *wire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return ""
	}

	return hex.EncodeToString(buf.Bytes())
}
