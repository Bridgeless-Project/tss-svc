package utils

import (
	"bytes"
	"encoding/hex"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

func FeeRateValid(fee btcutil.Amount) bool {
	if fee < DefaultFeeRateBtcPerKvb || fee > MaxFeeRateBtcPerKvb {
		return false
	}

	return true
}

func EncodeTransaction(tx *wire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return ""
	}

	return hex.EncodeToString(buf.Bytes())
}

func MapUnspent(unspent []btcjson.ListUnspentResult) map[types.OutPoint]btcjson.ListUnspentResult {
	unspentMap := make(map[types.OutPoint]btcjson.ListUnspentResult, len(unspent))
	for _, u := range unspent {
		unspentMap[types.OutPoint{TxID: u.TxID, Index: u.Vout}] = u
	}

	return unspentMap
}

func FindUsedInputs(tx wire.MsgTx, unspent []btcjson.ListUnspentResult) ([]btcjson.ListUnspentResult, error) {
	unspentMap := MapUnspent(unspent)

	usedInputs := make([]btcjson.ListUnspentResult, len(tx.TxIn))
	for i, txIn := range tx.TxIn {
		if txIn == nil {
			return nil, errors.Errorf("nil input at index %d", i)
		}

		outPoint := types.OutPoint{
			TxID:  txIn.PreviousOutPoint.Hash.String(),
			Index: txIn.PreviousOutPoint.Index,
		}
		out, found := unspentMap[outPoint]
		if !found {
			return nil, errors.Errorf("input %s:%d not found in unspent outputs", outPoint.TxID, outPoint.Index)
		}

		usedInputs[i] = out
	}

	return usedInputs, nil
}

func ToUnits(f float64) int64 {
	amt, err := btcutil.NewAmount(f)
	if err != nil {
		return 0
	}

	return int64(amt)
}
