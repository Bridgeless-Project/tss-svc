package operations

import (
	"bytes"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

type WithdrawOperationData struct {
	WithdrawalAmount *big.Int
	Receiver         common.Address
	TxHash           string
	TxNonce          int64
	ChainId          *big.Int
	DestinationToken common.Address
	IsWrapped        bool
}

func (w WithdrawOperationData) ToOperation() Operation {
	if w.DestinationToken == common.HexToAddress(bridge.DefaultNativeTokenAddress) {
		return &WithdrawNativeContent{
			Amount:  ToBytes32(w.WithdrawalAmount.Bytes()),
			TxHash:  TxHashToBytes32(w.TxHash),
			TxNonce: IntToBytes32(w.TxNonce),
			ChainID: ToBytes32(w.ChainId.Bytes()),
		}
	}

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

func NewWithdrawERC20Content(data db.Deposit) (Operation, error) {
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

	return WithdrawOperationData{
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

func (w WithdrawERC20Content) CalculateHashPrefixed() []byte {
	return SetSignaturePrefix(w.CalculateHash())
}

func (w WithdrawERC20Content) Equals(other []byte) bool {
	return bytes.Equal(other, w.CalculateHash())
}
