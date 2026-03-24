package solana

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"time"

	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	tsscommon "github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type ContractOperationUpdateSigner uint8

const (
	ContractOperationAddSigner    ContractOperationUpdateSigner = 1
	ContractOperationRemoveSigner ContractOperationUpdateSigner = 2
)

type UpdateSignerOperation struct {
	signer    *ecdsa.PublicKey
	startTime int64 // unix timestamp
	deadline  int64 // unix timestamp
	nonce     *big.Int

	bridgeId string
	opType   ContractOperationUpdateSigner
}

func (u UpdateSignerOperation) Signer() string {
	return hexutil.Encode(crypto.CompressPubkey(u.signer))
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
	return hexutil.Encode(append(sig.Signature, sig.SignatureRecovery...))
}

func (u UpdateSignerOperation) CalculateHash() []byte {
	signingData := append([]byte(u.bridgeId), crypto.CompressPubkey(u.signer)...)

	signingData = binary.LittleEndian.AppendUint64(signingData, uint64(u.startTime))
	signingData = binary.LittleEndian.AppendUint64(signingData, uint64(u.deadline))
	signingData = binary.LittleEndian.AppendUint64(signingData, u.nonce.Uint64())

	signingData = append(signingData, byte(u.opType))

	hash := sha256.Sum256(signingData)

	return hash[:]
}

func NewAddSignerOperation(
	signer *ecdsa.PublicKey,
	startTime time.Time,
	bridgeId string,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		signer:    signer,
		startTime: startTime.UTC().Unix(),
		deadline:  resharingTypes.OperationAddSignerDeadline(startTime).Unix(),
		nonce:     resharingTypes.OperationUpdateSignerNonce(startTime),
		bridgeId:  bridgeId,
		opType:    ContractOperationAddSigner,
	}
}

func NewRemoveSignerOperation(
	signer *ecdsa.PublicKey,
	startTime time.Time,
	bridgeId string,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		signer:    signer,
		startTime: resharingTypes.OperationRemoveSignerStartTime(startTime).Unix(),
		deadline:  resharingTypes.OperationRemoveSignerDeadline(startTime).Unix(),
		nonce:     resharingTypes.OperationUpdateSignerNonce(startTime),
		bridgeId:  bridgeId,
		opType:    ContractOperationRemoveSigner,
	}
}
