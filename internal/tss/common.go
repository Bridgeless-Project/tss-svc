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
	Format() SignatureFormat
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

type SignatureFormat string

const (
	SignatureFormatECDSARecoverable SignatureFormat = "ecdsa_recoverable"
	SignatureFormatSchnorrTaproot   SignatureFormat = "schnorr_taproot"
)

func RequireSignatureFormat(sig SignatureData, want SignatureFormat) error {
	if sig == nil {
		return fmt.Errorf("nil signature")
	}
	if sig.Format() != want {
		return fmt.Errorf("unsupported signature format %q, expected %q", sig.Format(), want)
	}

	return nil
}

func RecoverableECDSASignatureBytes(sig SignatureData) ([]byte, error) {
	if err := RequireSignatureFormat(sig, SignatureFormatECDSARecoverable); err != nil {
		return nil, err
	}

	r := sig.GetR()
	if len(r) != 32 {
		return nil, fmt.Errorf("invalid recoverable ECDSA R length: %d", len(r))
	}
	s := sig.GetS()
	if len(s) != 32 {
		return nil, fmt.Errorf("invalid recoverable ECDSA S length: %d", len(s))
	}
	recovery := sig.GetSignatureRecovery()
	if len(recovery) != 1 {
		return nil, fmt.Errorf("invalid recoverable ECDSA recovery length: %d", len(recovery))
	}

	result := make([]byte, 0, 65)
	result = append(result, r...)
	result = append(result, s...)
	result = append(result, recovery...)

	return result, nil
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

func (s *Signatures) SetSignature(data SignatureData) {
	s.Data = append(s.Data, data)
}

func MaxMaliciousParties(partiesCount, threshold int) int {
	// T+1 parties are required to function
	return partiesCount - (threshold + 1)
}
