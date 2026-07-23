package zano

import (
	"encoding/base64"
	"encoding/json"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/pkg/errors"
)

type DepositMemo struct {
	Address    string `json:"dst_add"`
	ChainId    string `json:"dst_net_id"`
	ReferralId uint16 `json:"referral_id,omitempty"`

	DestinationToken     string `json:"dst_token,omitempty"`
	MinDestinationAmount string `json:"min_dst_amount,omitempty"`
	SwapDeadline         string `json:"swap_deadline,omitempty"`
}

type SwapParams struct {
	MinDestinationAmount *big.Int
	SwapDeadline         *big.Int
}

func (m *DepositMemo) IsSwap() bool {
	return m.DestinationToken != ""
}

func (m *DepositMemo) Validate() error {
	if m.Address == "" {
		return errors.New("address is empty")
	}
	if m.ChainId == "" {
		return errors.New("chain id is empty")
	}

	if !m.IsSwap() {
		if m.MinDestinationAmount != "" || m.SwapDeadline != "" {
			return errors.New("swap fields set without destination token")
		}

		return nil
	}

	_, err := m.SwapParams()

	return err
}

func (m *DepositMemo) SwapParams() (*SwapParams, error) {
	minAmount, ok := new(big.Int).SetString(m.MinDestinationAmount, 10)
	if !ok {
		return nil, errors.New("invalid min destination amount")
	}
	if minAmount.Cmp(bridge.ZeroAmount) <= 0 {
		return nil, errors.New("min destination amount must be positive")
	}

	deadline, ok := new(big.Int).SetString(m.SwapDeadline, 10)
	if !ok {
		return nil, errors.New("invalid swap deadline")
	}
	if !deadline.IsInt64() || deadline.Cmp(bridge.ZeroAmount) <= 0 {
		return nil, errors.New("swap deadline out of range")
	}

	return &SwapParams{
		MinDestinationAmount: minAmount,
		SwapDeadline:         deadline,
	}, nil
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
