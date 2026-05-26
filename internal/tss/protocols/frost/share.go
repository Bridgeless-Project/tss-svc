package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/fxamacker/cbor/v2"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type FrostShare struct {
	data         *frost.Config
	protocolType tss.ProtocolType
	group        curve.Curve
}

type ECDSAShare struct {
	data         *ecdsa.LocalPartySaveData
	protocolType string
	group        curve.Curve
}

func NewFrostShare() *FrostShare {
	return &FrostShare{
		protocolType: protocolFROST,
	}
}

func (f *FrostShare) Protocol() tss.ProtocolType {
	return f.protocolType
}

func (f *FrostShare) MustFrostShare() *frost.Config {
	return f.data
}

func (f *FrostShare) MustEcdsaShare() *ecdsa.LocalPartySaveData {
	return nil
}

func (f *FrostShare) SetData(data any) error {
	config, ok := data.(*frost.Config)
	if !ok {
		return errors.New("invalid data type")
	}
	f.data = config
	return nil
}

func (f *FrostShare) Marshal() ([]byte, error) {
	data, err := cbor.Marshal(f.data)
	return data, err
}

func (f *FrostShare) Unmarshal(data []byte) error {
	config := frost.EmptyConfig(f.group)
	if err := cbor.Unmarshal(data, config); err != nil {
		return errors.Wrap(err, "failed to decode frost share data")
	}

	f.data = config

	return nil
}

func (f *FrostShare) Verify(signature, data []byte) (bool, error) {
	return false, nil
}

func (f *FrostShare) PubKey() []byte {
	return nil
}

func (f *FrostShare) SetVaultData(kvData map[string]interface{}) error {
	val, ok := kvData[valueVaultKey].(string)
	if !ok {
		return errors.New("share data not found")
	}

	data := frost.EmptyConfig(curve.Secp256k1{})
	if err := cbor.Unmarshal([]byte(val), data); err != nil {
		return errors.Wrap(err, "failed to decode frost share data")
	}

	f.data = data
	return nil
}

func (f *FrostShare) GetVaultPath() string {
	return "" // TODO: fix it
}

func (f *FrostShare) Group() curve.Curve {
	if f.group == nil {
		f.group = curve.Secp256k1{}
	}
	return f.group
}

func (f *FrostShare) WithGroup(g curve.Curve) *FrostShare {
	f.group = g
	return f
}
func (f *FrostShare) PreParams() tss.PreParams {
	return nil
}
