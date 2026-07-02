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

func (s *FrostSignature) Format() tss.SignatureFormat {
	return tss.SignatureFormatSchnorrTaproot
}

func (s *FrostSignature) SetSignature(signature any) error {
	rawSignature, ok := signature.([]byte)
	if !ok {
		return errors.Errorf("unexpected FROST signature type %T", signature)
	}
	if len(rawSignature) != taproot.SignatureLen {
		return errors.Errorf("invalid FROST signature length: %d", len(rawSignature))
	}

	s.data = append([]byte(nil), rawSignature...)
	s.r = nil
	s.s = nil
	s.m = nil

	return nil
}

func (s *FrostSignature) GetSignature() []byte {
	if s == nil {
		return nil
	}

	return s.data
}

func (s *FrostSignature) GetSignatureRecovery() []byte {
	return nil
}

func (s *FrostSignature) GetR() []byte {
	if s == nil {
		return nil
	}

	return s.r
}

func (s *FrostSignature) GetS() []byte {
	if s == nil {
		return nil
	}

	return s.s
}

func (s *FrostSignature) GetM() []byte {
	if s == nil {
		return nil
	}

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
