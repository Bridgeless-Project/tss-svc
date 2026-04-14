package ton

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
	"time"

	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	tsscommon "github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type UpdateSignerOperation struct {
	signer    *ecdsa.PublicKey
	startTime int64 // unix timestamp
	deadline  int64 // unix timestamp
	nonce     *big.Int
	isAdding  bool
}

func (u UpdateSignerOperation) Signer() string {
	return hexutil.Encode(elliptic.Marshal(tss.S256(), u.signer.X, u.signer.Y))
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
	builder := cell.BeginCell().
		MustStoreUInt(4, 8). // key recovery
		MustStoreBigUInt(u.signer.X, 256). // pubkey
		MustStoreBigUInt(u.signer.Y, 256).
		MustStoreInt(u.startTime, 64). // data
		MustStoreInt(u.deadline, 64).
		MustStoreInt(u.nonce.Int64(), 64).
		MustStoreBoolBit(u.isAdding). // operation type
		MustStoreUInt(0, 7) // padding to 8 bits

	raw := builder.EndCell().ToRawUnsafe()

	return crypto.Keccak256(raw.Data)
}

func NewAddSignerOperation(
	signer *ecdsa.PublicKey,
	startTime time.Time,
) UpdateSignerOperation {
	return UpdateSignerOperation{
		signer:    signer,
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
		signer:    signer,
		startTime: resharingTypes.OperationRemoveSignerStartTime(startTime).Unix(),
		deadline:  resharingTypes.OperationRemoveSignerDeadline(startTime).Unix(),
		nonce:     resharingTypes.OperationUpdateSignerNonce(startTime),
		isAdding:  false,
	}
}
