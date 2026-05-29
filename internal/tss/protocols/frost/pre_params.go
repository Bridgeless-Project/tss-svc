package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
)

type FrostPreParams struct{}

func NewFrostPreParams() *FrostPreParams {
	return &FrostPreParams{}
}

func (f *FrostPreParams) Protocol() tss.ProtocolType {
	return protocolFROST
}

func (f *FrostPreParams) MustEcdsaPreParams() ecdsa.LocalPreParams {
	return ecdsa.LocalPreParams{}
}

func (f *FrostPreParams) MustFrostPreParams() *frost.Config {
	return nil
}

func (f *FrostPreParams) SetData(data any) error {
	return nil
}

func (f *FrostPreParams) Marshal() ([]byte, error) {
	return nil, nil
}

func (f *FrostPreParams) Unmarshal([]byte) error {
	return nil
}

func (f *FrostPreParams) GetVaultPath() string {
	return ""
}

func (f *FrostPreParams) SetVaultData(map[string]interface{}) error {
	return nil
}
