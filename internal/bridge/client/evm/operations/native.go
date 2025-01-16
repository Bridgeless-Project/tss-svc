package operations

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"math/big"
)

type WithdrawNativeContent struct {
	Amount   []byte
	Receiver []byte
	TxHash   []byte
	TxNonce  []byte
	ChainID  []byte
}

func NewWithdrawNativeContent(event db.DepositData) (*WithdrawNativeContent, error) {
	destinationChainID, ok := new(big.Int).SetString(event.DestinationChainId, 10)
	if !ok {
		return nil, errors.New("invalid chain id")
	}

	return &WithdrawNativeContent{
		Amount:   ToBytes32(event.WithdrawalAmount.Bytes()),
		Receiver: hexutil.MustDecode(event.DestinationAddress),
		TxHash:   hexutil.MustDecode(event.TxHash),
		TxNonce:  IntToBytes32(event.TxEventId),
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
