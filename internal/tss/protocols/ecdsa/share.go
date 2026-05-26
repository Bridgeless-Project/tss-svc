package tss

import (
	"encoding/json"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/fxamacker/cbor/v2"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type EcdsaShare struct {
	data         *ecdsa.LocalPartySaveData
	protocolType tss.ProtocolType
	group        curve.Curve

	preparams tss.PreParams
}

func NewEcdsaShare() *EcdsaShare {
	return &EcdsaShare{
		protocolType: protocolECDSA,
	}
}

func (e *EcdsaShare) Protocol() tss.ProtocolType {
	return e.protocolType
}

func (e *EcdsaShare) MustFrostShare() *frost.Config {
	return nil
}

func (e *EcdsaShare) MustEcdsaShare() *ecdsa.LocalPartySaveData {
	return e.data
}

func (e *EcdsaShare) SetData(data any) error {
	config, ok := data.(*ecdsa.LocalPartySaveData)
	if !ok {
		return errors.New("invalid data type")
	}
	e.data = config
	return nil
}

func (e *EcdsaShare) Marshal() ([]byte, error) {
	data, err := cbor.Marshal(e.data)
	return data, err
}

func (e *EcdsaShare) Unmarshal(data []byte) error {
	if err := cbor.Unmarshal(data, e.data); err != nil {
		return errors.Wrap(err, "failed to decode frost share data")
	}

	return nil
}

func (e *EcdsaShare) Verify(signature, data []byte) (bool, error) {
	return false, nil
}

func (e *EcdsaShare) PubKey() []byte {
	return nil
}

func (e *EcdsaShare) SetVaultData(kvData map[string]interface{}) error {
	val, ok := kvData[valueVaultKey].(string)
	if !ok {
		return errors.New("share data not found")
	}
	data := new(keygen.LocalPartySaveData)
	if err := json.Unmarshal([]byte(val), data); err != nil {
		return errors.Wrap(err, "failed to decode share data")
	}
	e.data = data

	return nil

}

func (e *EcdsaShare) GetVaultPath() string {
	return ""
}

func (e *EcdsaShare) Group() curve.Curve {
	if e.group == nil {
		e.group = curve.Secp256k1{}
	}

	return e.group
}

func (e *EcdsaShare) PreParams() tss.PreParams {
	return e.preparams
}
