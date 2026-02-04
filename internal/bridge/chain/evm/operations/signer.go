package operations

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type UpdateSignerOperation struct {
	Signer    common.Address
	StartTime int64 // unix timestamp
	Deadline  int64 // unix timestamp
	Nonce     *big.Int
	IsAdding  bool
}

func NewAddSignerOperation(
	signer common.Address,
	startTime time.Time,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		Signer:    signer,
		StartTime: startTime.UTC().Unix(),
		Deadline:  CalculateAddSignerOperationDeadline(startTime).Unix(),
		Nonce:     UpdateSignerOperationNonce(startTime),
		IsAdding:  true,
	}
}

func NewRemoveSignerOperation(
	signer common.Address,
	startTime time.Time,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		Signer:    signer,
		StartTime: startTime.UTC().Unix(),
		Deadline:  CalculateRemoveSignerOperationDeadline(startTime).Unix(),
		Nonce:     UpdateSignerOperationNonce(startTime),
		IsAdding:  false,
	}
}

func (u UpdateSignerOperation) CalculateHash() []byte {
	return crypto.Keccak256(
		u.Signer.Bytes(),
		IntToBytes32(u.StartTime),
		IntToBytes32(u.Deadline),
		ToBytes32(u.Nonce.Bytes()),
		BoolToBytes(u.IsAdding),
	)
}

func CalculateRemoveSignerOperationDeadline(startTime time.Time) time.Time {
	return startTime.Add(7 * 24 * time.Hour).UTC()
}

func CalculateAddSignerOperationDeadline(startTime time.Time) time.Time {
	return startTime.Add(24 * time.Hour).UTC()
}

func UpdateSignerOperationNonce(startTime time.Time) *big.Int {
	return big.NewInt(startTime.UTC().Unix())
}
