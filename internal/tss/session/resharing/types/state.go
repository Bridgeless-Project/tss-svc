package types

import (
	"crypto/ecdsa"
	"sync"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
)

type State struct {
	GlobalStartTime  time.Time
	SessionStartTime time.Time

	Epoch     uint32
	NewPubKey *ecdsa.PublicKey
	Threshold uint

	Account  *core.Account
	OldShare *keygen.LocalPartySaveData
	NewShare *keygen.LocalPartySaveData

	Signatures         []bridgeTypes.EpochChainSignatures
	NewBridgeAddresses map[string]string // chainId -> addr

	mu sync.Mutex
}

func InitializeState(
	epoch uint32,
	threshold uint,
	globalStartTime time.Time,
	account *core.Account,
) *State {
	return &State{
		Epoch:              epoch,
		Threshold:          threshold,
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

func (s *State) AddSignature(signature bridgeTypes.EpochChainSignatures) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Signatures = append(s.Signatures, signature)
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
