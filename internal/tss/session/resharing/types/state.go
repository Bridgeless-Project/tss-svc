package types

import (
	"crypto/ecdsa"
	"time"
)

type State struct {
	Epoch              uint32
	GlobalStartTime    time.Time
	SessionStartTime   time.Time
	NewPubKey          *ecdsa.PublicKey
	EvmData            EvmData
	NewBridgeAddresses map[string]string // chainId -> addr
}

func InitializeState(globalStartTime time.Time) *State {
	return &State{
		GlobalStartTime:    globalStartTime,
		NewBridgeAddresses: make(map[string]string),
	}
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
