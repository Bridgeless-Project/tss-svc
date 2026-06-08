package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
)

var _ tss.SignatureData = new(FrostSignature)

type FrostSignature struct {
	data []byte
	m    []byte
	r    []byte
	s    []byte
}

func (s *FrostSignature) SetSignature(signature any) error {
	s.data = signature.([]byte)

	if len(s.data) == taproot.SignatureLen {
		s.r = s.data[:32]
		s.s = s.data[32:]
	}

	return nil
}

func (s *FrostSignature) GetSignature() []byte {
	return s.data
}

func (s *FrostSignature) GetSignatureRecovery() []byte {
	return nil
}

func (s *FrostSignature) GetR() []byte {
	return s.r
}

func (s *FrostSignature) GetS() []byte {
	return s.s
}

func (s *FrostSignature) GetM() []byte {
	return s.m
}
