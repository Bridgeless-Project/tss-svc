package tss

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v3/common"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	ecdsaKeygen "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fxamacker/cbor/v2"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

const keyShare = "tss_shares/ecdsa"

type EcdsaShare struct {
	data         *ecdsaKeygen.LocalPartySaveData
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

func (e *EcdsaShare) MustEcdsaShare() *ecdsaKeygen.LocalPartySaveData {
	return e.data
}

func (e *EcdsaShare) SetData(data any) error {
	config, ok := data.(*ecdsaKeygen.LocalPartySaveData)
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
	e.data = new(ecdsaKeygen.LocalPartySaveData)
	if err := cbor.Unmarshal(data, e.data); err != nil {
		return errors.Wrap(err, "failed to decode ecdsa share data")
	}

	return nil
}

func (e *EcdsaShare) Verify(signature, data []byte) (bool, error) {
	if e.data == nil || e.data.ECDSAPub == nil {
		return false, errors.New("public key is not set")
	}
	if len(signature) == 0 {
		return false, errors.New("signature is invalid")
	}

	sigData := &common.SignatureData{Signature: signature}
	if len(signature) >= 64 {
		sigData.R = append([]byte(nil), signature[:32]...)
		sigData.S = append([]byte(nil), signature[32:64]...)
	}

	return verifyEcdsaSignature(e.data.ECDSAPub.ToECDSAPubKey(), data, sigData), nil
}

func (e *EcdsaShare) PubKey() []byte {
	if e.data == nil || e.data.ECDSAPub == nil {
		return nil
	}

	return crypto.CompressPubkey(e.data.ECDSAPub.ToECDSAPubKey())
}

func verifyEcdsaSignature(pubKey *ecdsa.PublicKey, data []byte, sig *common.SignatureData) bool {
	if pubKey == nil || sig == nil || len(sig.R) == 0 || len(sig.S) == 0 {
		return false
	}

	r, s := new(big.Int).SetBytes(sig.GetR()), new(big.Int).SetBytes(sig.GetS())
	return ecdsa.Verify(pubKey, data, r, s)
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
	return keyShare
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
