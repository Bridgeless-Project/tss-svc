package utxo

import (
	"bytes"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
)

func EncodeTransaction(tx *wire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return ""
	}

	return hex.EncodeToString(buf.Bytes())
}

func ToUnits(f float64) int64 {
	amt, err := btcutil.NewAmount(f)
	if err != nil {
		return 0
	}

	return int64(amt)
}
