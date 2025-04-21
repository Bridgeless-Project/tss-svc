package operations

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

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

	return &WithdrawERC20Content{
		Amount:                  ToBytes32(withdrawalAmount.Bytes()),
		Receiver:                hexutil.MustDecode(data.Receiver),
		TxHash:                  hexutil.MustDecode(data.TxHash),
		TxNonce:                 IntToBytes32(data.TxNonce),
		ChainID:                 ToBytes32(destinationChainID.Bytes()),
		DestinationTokenAddress: common.HexToAddress(data.WithdrawalToken).Bytes(),
		IsWrapped:               BoolToBytes(data.IsWrappedToken),
	}, nil
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
