package operations

import (
	"bytes"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

type WithdrawERC20OperationData struct {
	WithdrawalAmount *big.Int
	Receiver         common.Address
	TxHash           string
	TxNonce          int64
	ChainId          *big.Int
	DestinationToken common.Address
	IsWrapped        bool
}

func (w WithdrawERC20OperationData) ToOperation() *WithdrawERC20Content {
	return &WithdrawERC20Content{
		Amount:                  ToBytes32(w.WithdrawalAmount.Bytes()),
		Receiver:                w.Receiver.Bytes(),
		TxHash:                  TxHashToBytes32(w.TxHash),
		TxNonce:                 IntToBytes32(w.TxNonce),
		ChainID:                 ToBytes32(w.ChainId.Bytes()),
		DestinationTokenAddress: w.DestinationToken.Bytes(),
		IsWrapped:               BoolToBytes(w.IsWrapped),
	}
}

type WithdrawERC20Content struct {
	DestinationTokenAddress []byte
	Amount                  []byte
	Receiver                []byte
	TxHash                  []byte
	TxNonce                 []byte
	ChainID                 []byte
	IsWrapped               []byte
}

func NewWithdrawERC20Content(data db.Deposit) (*WithdrawERC20Content, error) {
	destinationChainID, ok := new(big.Int).SetString(data.WithdrawalChainId, 10)
	if !ok {
		return nil, errors.New("invalid chain id")
	}

	withdrawalAmount, ok := new(big.Int).SetString(data.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("invalid withdrawal amount")
	}

	if !common.IsHexAddress(data.Receiver) {
		return nil, errors.New("invalid destination address")
	}
	if !common.IsHexAddress(data.WithdrawalToken) {
		return nil, errors.New("invalid destination token address")
	}

	return WithdrawERC20OperationData{
		WithdrawalAmount: withdrawalAmount,
		Receiver:         common.HexToAddress(data.Receiver),
		TxHash:           data.TxHash,
		TxNonce:          data.TxNonce,
		ChainId:          destinationChainID,
		DestinationToken: common.HexToAddress(data.WithdrawalToken),
		IsWrapped:        data.IsWrappedToken,
	}.ToOperation(), nil
}

func (w WithdrawERC20Content) CalculateHash() []byte {
	return crypto.Keccak256(
		w.DestinationTokenAddress,
		w.Amount,
		w.Receiver,
		w.TxHash,
		w.TxNonce,
		w.ChainID,
		w.IsWrapped,
	)
}

func (w WithdrawERC20Content) Equals(other []byte) bool {
	return bytes.Equal(other, w.CalculateHash())
}
