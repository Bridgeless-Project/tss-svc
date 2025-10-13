package zano

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

type DepositMemo struct {
	Address    string `json:"dst_add"`
	ChainId    string `json:"dst_net_id"`
	ReferralId uint16 `json:"referral_id,omitempty"`
}

func (m *DepositMemo) Validate() error {
	if m.Address == "" {
		return errors.New("address is empty")
	}
	if m.ChainId == "" {
		return errors.New("chain id is empty")
	}

	return nil
}

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
