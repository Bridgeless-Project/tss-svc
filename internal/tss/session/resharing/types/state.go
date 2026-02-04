package types

import (
	"crypto/ecdsa"
	"time"
)

type State struct {
	GlobalStartTime  time.Time
	SessionStartTime time.Time
	NewPubKey        *ecdsa.PublicKey
	EvmData          EvmData
}

// TODO: consider defining on Core
type UpdateSignerEvmSignature struct {
	Signer    string
	StartTime int64
	Deadline  int64
	Nonce     uint64
	Signature string
}

type EvmData struct {
	AddNewSignerSignature    UpdateSignerEvmSignature
	RemoveOldSignerSignature UpdateSignerEvmSignature
}
