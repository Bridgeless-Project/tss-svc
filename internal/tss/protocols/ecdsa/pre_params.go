package tss

import (
	"encoding/json"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

const keyPreParams = "keygen_preparams/ecdsa"

type EcdsaPreParams struct {
	data         *keygen.LocalPreParams
	protocolType tss.ProtocolType
}

func NewEcdsaPreParams() *EcdsaPreParams {
	return &EcdsaPreParams{
		protocolType: protocolECDSA,
	}
}

func (f *EcdsaPreParams) Protocol() tss.ProtocolType {
	return f.protocolType
}

func (f *EcdsaPreParams) MustFrostPreParams() *frost.Config {
	return nil
}

func (f *EcdsaPreParams) MustEcdsaPreParams() keygen.LocalPreParams {
	if f.data == nil {
		return keygen.LocalPreParams{}
	}

	return *f.data
}

func (f *EcdsaPreParams) SetData(data any) error {
	config, ok := data.(*keygen.LocalPreParams)
	if !ok {
		return errors.New("invalid data type")
	}
	f.data = config
	return nil
}

func (f *EcdsaPreParams) Marshal() ([]byte, error) {
	return json.Marshal(f.data)
}

func (f *EcdsaPreParams) Unmarshal(data []byte) error {
	f.data = new(keygen.LocalPreParams)
	if err := json.Unmarshal(data, f.data); err != nil {
		return errors.Wrap(err, "failed to decode ecdsa preparams data")
	}

	return nil
}

func (f *EcdsaPreParams) SetVaultData(kvData map[string]interface{}) error {
	val, ok := kvData[valueVaultKey].(string)
	if !ok {
		return errors.New("share data not found")
	}

	data := new(keygen.LocalPreParams)
	if err := json.Unmarshal([]byte(val), data); err != nil {
		return errors.Wrap(err, "failed to decode share data")
	}

	f.data = data

	return nil

}

func (f *EcdsaPreParams) GetVaultPath() string {
	return keyPreParams
}
