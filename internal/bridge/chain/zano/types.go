package zano

import (
	"encoding/base64"
	"encoding/json"
)

type SignedTransaction struct {
	UnsignedTransaction
	Signature string
}

type UnsignedTransaction struct {
	ExpectedTxHash string
	FinalizedTx    string
	Data           string
}

func (tx *SignedTransaction) Encode() string {
	raw, _ := json.Marshal(tx)

	return base64.StdEncoding.EncodeToString(raw)
}
