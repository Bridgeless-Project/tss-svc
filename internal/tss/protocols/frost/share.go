package tss

import (
	"encoding/base64"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/fxamacker/cbor/v2"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

const keyShare = "tss_shares/frost"

type FrostShare struct {
	data         *frost.Config
	protocolType tss.ProtocolType
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

// We need to use CBOR to prevent data loss
func (f *FrostShare) Marshal() ([]byte, error) {
	if f.data == nil {
		return nil, errors.New("missing frost share")
	}

	return cbor.Marshal(f.data)
}

func (f *FrostShare) Unmarshal(raw []byte) error {
	config := frost.EmptyConfig(f.Group())

	if err := cbor.Unmarshal(raw, config); err != nil {
		return errors.Wrap(err, "failed to decode frost share data")
	}

	f.data = config
	return nil
}

// TODO: do not use taproot for ZCash
func (f *FrostShare) Verify(signature, data []byte) (bool, error) {
	if len(signature) != taproot.SignatureLen {
		return false, errors.New("signature is invalid")
	}

	pubKey := f.PubKey()
	if len(pubKey) == 0 {
		return false, errors.New("public key is not set")
	}

	return taproot.PublicKey(pubKey).Verify(signature, data), nil
}

func (f *FrostShare) PubKey() []byte {
	if f.data == nil {
		return nil
	}

	publicKey, ok := f.data.PublicKey.(*curve.Secp256k1Point)
	if !ok {
		return nil
	}

	return append([]byte(nil), publicKey.XBytes()...)
}

func (f *FrostShare) SetVaultData(kvData map[string]interface{}) error {
	val, ok := kvData[valueVaultKey].(string)
	if !ok {
		return errors.New("share data not found")
	}

	raw, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return errors.Wrap(err, "failed to base64 decode frost share data")
	}

	err = f.Unmarshal(raw)
	if err != nil {
		return errors.Wrap(err, "failed to decode frost share data")
	}

	return nil
}

func (f *FrostShare) GetVaultPath() string {
	return keyShare
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
