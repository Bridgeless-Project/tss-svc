package tss

import (
	"encoding/gob"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
)

var _ tss.SignatureData = new(FrostSignature)

func init() {
	gob.Register(new(FrostSignature))
}

type FrostSignature struct {
	data []byte
	m    []byte
	r    []byte
	s    []byte
}

func (s *FrostSignature) SetSignature(signature any) error {
	rawSignature, ok := signature.([]byte)
	if !ok {
		return errors.Errorf("unexpected FROST signature type %T", signature)
	}
	s.data = append([]byte(nil), rawSignature...)
	s.r = nil
	s.s = nil

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

func (s *FrostSignature) GobEncode() ([]byte, error) {
	if s == nil || len(s.data) == 0 {
		return nil, errors.New("missing FROST signature data")
	}

	return append([]byte(nil), s.data...), nil
}

func (s *FrostSignature) GobDecode(data []byte) error {
	return s.SetSignature(data)
}
