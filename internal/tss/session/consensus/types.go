package consensus

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"google.golang.org/protobuf/proto"
)

var _ SigningData = &PlainSignData{}

type SignStartData struct {
	*p2p.SignStartData
}

func (s SignStartData) HashString() string {
	if s.SignStartData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.SignStartData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type SigningData interface {
	broadcast.Hashable
	SignHashes() [][]byte
}

type Mechanism[T SigningData] interface {
	FormProposalData() (*T, error)
	VerifyProposedData(T) error
}

type PlainSignData struct {
	signHash []byte
}

func NewPlainSignData(signHash []byte) *PlainSignData {
	return &PlainSignData{signHash: signHash}
}

func (p PlainSignData) HashString() string {
	return fmt.Sprintf("%x", sha256.Sum256(p.signHash))
}

func (p PlainSignData) SignHashes() [][]byte {
	return [][]byte{p.signHash}
}

type PlainSignDataMechanism struct {
	SignHash []byte
}

func NewPlainSignDataMechanism(signHash []byte) *PlainSignDataMechanism {
	return &PlainSignDataMechanism{SignHash: signHash}
}

func (m *PlainSignDataMechanism) FormProposalData() (*PlainSignData, error) {
	return NewPlainSignData(m.SignHash), nil
}

func (m *PlainSignDataMechanism) VerifyProposedData(data PlainSignData) error {
	if !bytes.Equal(m.SignHash, data.signHash) {
		return fmt.Errorf("sign hash mismatch")
	}

	return nil
}
