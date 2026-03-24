package types

import (
	"math/big"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
)

type ContractOperation interface {
	Signer() string
	StartTime() time.Time
	Deadline() time.Time
	Nonce() uint64

	CalculateHash() []byte
	ConvertSignature(sig *common.SignatureData) string
}

func OperationRemoveSignerStartTime(startTime time.Time) time.Time {
	return startTime.Add(14 * 24 * time.Hour).UTC()
}

func OperationRemoveSignerDeadline(startTime time.Time) time.Time {
	return OperationRemoveSignerStartTime(startTime).Add(3 * 24 * time.Hour).UTC()
}

func OperationAddSignerDeadline(startTime time.Time) time.Time {
	return startTime.Add(24 * time.Hour).UTC()
}

func OperationUpdateSignerNonce(startTime time.Time) *big.Int {
	return big.NewInt(startTime.UTC().Unix())
}
