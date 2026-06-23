package tss

import (
	"encoding/gob"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	tsslib "github.com/bnb-chain/tss-lib/v3/common"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var _ tss.SignatureData = new(EcdsaSignature)

func init() {
	gob.Register(new(EcdsaSignature))
}

type EcdsaSignature struct {
	data *tsslib.SignatureData
}

func (s *EcdsaSignature) SetSignature(signature any) error {
	sigData, ok := signature.(*tsslib.SignatureData)
	if !ok {
		return errors.Errorf("unexpected ECDSA signature type %T", signature)
	}
	s.data = sigData
	return nil
}

func (s *EcdsaSignature) GetSignature() []byte {
	data, _ := proto.MarshalOptions{Deterministic: true}.Marshal(s.data)
	return data
}

func (s *EcdsaSignature) GetSignatureRecovery() []byte {
	return s.data.GetSignatureRecovery()
}

func (s *EcdsaSignature) GetR() []byte {
	return s.data.GetR()
}

func (s *EcdsaSignature) GetS() []byte {
	return s.data.GetS()
}
func (s *EcdsaSignature) GetM() []byte {
	return s.data.GetM()
}

func (s *EcdsaSignature) GobEncode() ([]byte, error) {
	if s == nil || s.data == nil {
		return nil, errors.New("missing ECDSA signature data")
	}

	return proto.MarshalOptions{Deterministic: true}.Marshal(s.data)
}

func (s *EcdsaSignature) GobDecode(data []byte) error {
	signature := new(tsslib.SignatureData)
	if err := proto.Unmarshal(data, signature); err != nil {
		return errors.Wrap(err, "failed to decode ECDSA signature data")
	}

	s.data = signature
	return nil
}
