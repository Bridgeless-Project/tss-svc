package tss

import (
	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
)

type ProtocolType string

type Share interface {
	Protocol() ProtocolType
	MustEcdsaShare() *ecdsa.LocalPartySaveData
	MustFrostShare() *frost.Config

	SetData(data any) error
	Marshal() ([]byte, error)
	Unmarshal([]byte) error

	GetVaultPath() string
	SetVaultData(kvData map[string]interface{}) error

	Verify(signature, data []byte) (bool, error)
	PubKey() []byte
	Group() curve.Curve

	PreParams() PreParams
}

type PreParams interface {
	Protocol() ProtocolType
	MustEcdsaPreParams() ecdsa.LocalPreParams
	MustFrostPreParams() frost.Config
	SetData(data any) error

	Marshal() ([]byte, error)
	Unmarshal([]byte) error

	GetVaultPath() string
	SetVaultData(kvData map[string]interface{}) error
}
