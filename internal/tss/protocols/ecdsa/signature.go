package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	tsslib "github.com/bnb-chain/tss-lib/v3/common"
	"google.golang.org/protobuf/proto"
)

var _ tss.SignatureData = new(EcdsaSignature)

type EcdsaSignature struct {
	data *tsslib.SignatureData
}

func (s EcdsaSignature) SetSignature(signature any) error {
	s.data = signature.(*tsslib.SignatureData)
	return nil
}

func (s EcdsaSignature) GetSignature() []byte {
	data, _ := proto.MarshalOptions{Deterministic: true}.Marshal(s.data)
	return data
}

func (s EcdsaSignature) GetSignatureRecovery() []byte {
	return s.data.GetSignatureRecovery()
}

func (s EcdsaSignature) GetR() []byte {
	return s.data.GetR()
}

func (s EcdsaSignature) GetS() []byte {
	return s.data.GetS()
}
func (s EcdsaSignature) GetM() []byte {
	return s.data.GetM()
}
