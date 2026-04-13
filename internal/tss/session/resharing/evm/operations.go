package evm

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	tsscommon "github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var _ resharingTypes.ContractOperation = UpdateSignerOperation{}

type UpdateSignerOperation struct {
	signer    common.Address
	startTime int64 // unix timestamp
	deadline  int64 // unix timestamp
	nonce     *big.Int
	isAdding  bool
}

func (u UpdateSignerOperation) Signer() string {
	return u.signer.Hex()
}

func (u UpdateSignerOperation) StartTime() time.Time {
	return time.Unix(u.startTime, 0).UTC()
}

func (u UpdateSignerOperation) Deadline() time.Time {
	return time.Unix(u.deadline, 0).UTC()
}

func (u UpdateSignerOperation) Nonce() uint64 {
	return u.nonce.Uint64()
}

func (u UpdateSignerOperation) ConvertSignature(sig *tsscommon.SignatureData) string {
	return evm.ConvertSignature(sig)
}

func (u UpdateSignerOperation) CalculateHash() []byte {
	return operations.SetSignaturePrefix(
		crypto.Keccak256(
			u.signer.Bytes(),
			operations.IntToBytes32(u.startTime),
			operations.IntToBytes32(u.deadline),
			operations.ToBytes32(u.nonce.Bytes()),
			operations.BoolToBytes(u.isAdding),
		),
	)
}

func NewAddSignerOperation(
	signer *ecdsa.PublicKey,
	startTime time.Time,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		signer:    evm.PubkeyToAddress(signer.X, signer.Y),
		startTime: startTime.UTC().Unix(),
		deadline:  resharingTypes.OperationAddSignerDeadline(startTime).Unix(),
		nonce:     resharingTypes.OperationUpdateSignerNonce(startTime),
		isAdding:  true,
	}
}

func NewRemoveSignerOperation(
	signer *ecdsa.PublicKey,
	startTime time.Time,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		signer:    evm.PubkeyToAddress(signer.X, signer.Y),
		startTime: resharingTypes.OperationRemoveSignerStartTime(startTime).Unix(),
		deadline:  resharingTypes.OperationRemoveSignerDeadline(startTime).Unix(),
		nonce:     resharingTypes.OperationUpdateSignerNonce(startTime),
		isAdding:  false,
	}
}
