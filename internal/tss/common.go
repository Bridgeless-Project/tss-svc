package tss

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	tsscommon "github.com/bnb-chain/tss-lib/v3/common"
)

const (
	OutChannelSize = 1000
	EndChannelSize = 1
	MsgsCapacity   = 100
)

func init() {
	// timing side-channel protection
	tsscommon.EnableConstantTimeOps()
}

type LocalKeygenParty struct {
	PreParams PreParams
	Address   core.Address
	Threshold int
}

type PartyMsg struct {
	Sender      core.Address
	WireMsg     []byte
	IsBroadcast bool
}

type SignatureData interface {
	SetSignature(signature any) error
	GetSignature() []byte
	GetSignatureRecovery() []byte
	GetR() []byte
	GetS() []byte
	GetM() []byte
}

type Signatures struct {
	Data []SignatureData
}

func (s Signatures) HashString() string {
	if len(s.Data) == 0 {
		return ""
	}

	var buff bytes.Buffer

	for _, sig := range s.Data {
		if sig == nil {
			continue
		}

		buff.Write(sig.GetSignature())
	}

	return fmt.Sprintf("%x", sha256.Sum256(buff.Bytes()))
}

func (s Signatures) SetSignature(data SignatureData) {
	s.Data = append(s.Data, data)
}

func MaxMaliciousParties(partiesCount, threshold int) int {
	// T+1 parties are required to function
	return partiesCount - (threshold + 1)
}
