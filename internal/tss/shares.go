package tss

import (
	"fmt"

	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
)

type ProtocolType string

const (
	ProtocolID_ECDSA ProtocolType = "ecdsa"
	ProtocolID_FROST ProtocolType = "frost"
)

func ValidateResharingProtocol(protocol ProtocolType) error {
	if protocol != ProtocolID_ECDSA {
		return fmt.Errorf("resharing currently supports only ECDSA, got %s", protocol)
	}
	return nil
}

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
	MustFrostPreParams() *frost.Config
	SetData(data any) error

	Marshal() ([]byte, error)
	Unmarshal([]byte) error

	GetVaultPath() string
	SetVaultData(kvData map[string]interface{}) error
}
