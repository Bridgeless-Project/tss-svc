package operations

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"math/big"
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

func NewWithdrawERC20Content(event db.DepositData) (*WithdrawERC20Content, error) {
	destinationChainID, ok := new(big.Int).SetString(event.DestinationChainId, 10)
	if !ok {
		return nil, errors.New("invalid chain id")
	}

	if !common.IsHexAddress(event.DestinationTokenAddress) {
		return nil, errors.New("invalid destination address")
	}

	return &WithdrawERC20Content{
		Amount:                  ToBytes32(event.WithdrawalAmount.Bytes()),
		Receiver:                hexutil.MustDecode(event.DestinationAddress),
		TxHash:                  hexutil.MustDecode(event.TxHash),
		TxNonce:                 IntToBytes32(event.TxEventId),
		ChainID:                 ToBytes32(destinationChainID.Bytes()),
		DestinationTokenAddress: common.HexToAddress(event.DestinationTokenAddress).Bytes(),
		IsWrapped:               BoolToBytes(event.IsWrappedToken),
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
