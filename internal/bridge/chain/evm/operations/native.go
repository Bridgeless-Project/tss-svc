package operations

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
)

type WithdrawNativeContent struct {
	Amount   []byte
	Receiver []byte
	TxHash   []byte
	TxNonce  []byte
	ChainID  []byte
}

func NewWithdrawNativeContent(data db.Deposit) (*WithdrawNativeContent, error) {
	destinationChainID, ok := new(big.Int).SetString(data.WithdrawalChainId, 10)
	if !ok {
		return nil, errors.New("invalid chain id")
	}

	withdrawalAmount, ok := new(big.Int).SetString(data.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("invalid withdrawal amount")
	}

	return &WithdrawNativeContent{
		Amount:   ToBytes32(withdrawalAmount.Bytes()),
		Receiver: hexutil.MustDecode(data.Receiver),
		TxHash:   hexutil.MustDecode(data.TxHash),
		TxNonce:  IntToBytes32(data.TxNonce),
		ChainID:  ToBytes32(destinationChainID.Bytes()),
	}, nil
}

func (w WithdrawNativeContent) CalculateHash() []byte {
	return crypto.Keccak256(
		w.Amount,
		w.Receiver,
		w.TxHash,
		w.TxNonce,
		w.ChainID,
	)
}

func (w WithdrawNativeContent) Equals(other []byte) bool {
	return bytes.Equal(other, w.CalculateHash())
}
