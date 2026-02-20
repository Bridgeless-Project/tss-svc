package types

import (
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
)

type State struct {
	GlobalStartTime  time.Time
	SessionStartTime time.Time

	Epoch     uint32
	NewPubKey *ecdsa.PublicKey
	Threshold uint

	EvmData            *EvmData
	NewBridgeAddresses map[string]string // chainId -> addr

	Account  *core.Account
	OldShare *keygen.LocalPartySaveData
	NewShare *keygen.LocalPartySaveData

	mu sync.Mutex
}

func InitializeState(
	epoch uint32,
	globalStartTime time.Time,
	account *core.Account,
) *State {
	return &State{
		Epoch:              epoch,
		GlobalStartTime:    globalStartTime,
		Account:            account,
		NewBridgeAddresses: make(map[string]string),
	}
}

func (s *State) AddBridgeAddress(chainId, addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.NewBridgeAddresses[chainId] = addr
}

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
