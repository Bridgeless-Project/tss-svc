package tss

import (
	"encoding/base64"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsa "github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/fxamacker/cbor/v2"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/math/polynomial"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
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

// Validate checks that a loaded share belongs to this node and consortium and
// that its private share, verification commitments, and group public key are
// internally consistent. Loading valid CBOR alone is not sufficient.
func (f *FrostShare) Validate(localID party.ID, threshold int, participants []party.ID) error {
	if f == nil || f.data == nil {
		return errors.New("missing frost share")
	}
	config := f.data
	if config.ID != localID {
		return fmt.Errorf("frost share belongs to %s, expected %s", config.ID, localID)
	}
	if threshold < 0 || threshold >= len(participants) {
		return fmt.Errorf("invalid threshold %d for %d FROST participants", threshold, len(participants))
	}
	if config.Threshold != threshold {
		return fmt.Errorf("frost share threshold is %d, expected %d", config.Threshold, threshold)
	}

	privateShare, ok := config.PrivateShare.(*curve.Secp256k1Scalar)
	if !ok || privateShare == nil || privateShare.IsZero() {
		return errors.New("invalid FROST secp256k1 private share")
	}
	publicKey, ok := config.PublicKey.(*curve.Secp256k1Point)
	if !ok || publicKey == nil || publicKey.IsIdentity() {
		return errors.New("invalid FROST secp256k1 public key")
	}
	if len(config.ChainKey) != 0 && len(config.ChainKey) != 32 {
		return fmt.Errorf("invalid FROST chain key length: %d", len(config.ChainKey))
	}
	if config.VerificationShares == nil || config.VerificationShares.Points == nil {
		return errors.New("missing FROST verification shares")
	}

	participantSet := make(map[party.ID]struct{}, len(participants))
	for _, id := range participants {
		if id == "" {
			return errors.New("empty FROST participant id")
		}
		if _, exists := participantSet[id]; exists {
			return fmt.Errorf("duplicate FROST participant id %s", id)
		}
		participantSet[id] = struct{}{}
		point, exists := config.VerificationShares.Points[id]
		if !exists || point == nil {
			return fmt.Errorf("missing FROST verification share for %s", id)
		}
		if _, ok := point.(*curve.Secp256k1Point); !ok {
			return fmt.Errorf("invalid FROST verification share curve for %s", id)
		}
	}
	if len(config.VerificationShares.Points) != len(participantSet) {
		return fmt.Errorf("FROST share contains %d verification shares, expected %d", len(config.VerificationShares.Points), len(participantSet))
	}
	for id := range config.VerificationShares.Points {
		if _, exists := participantSet[id]; !exists {
			return fmt.Errorf("unexpected FROST verification share for %s", id)
		}
	}
	if !privateShare.ActOnBase().Equal(config.VerificationShares.Points[localID]) {
		return errors.New("FROST private share does not match its verification share")
	}

	ordered := party.NewIDSlice(append([]party.ID(nil), participants...))
	quorum := ordered[:threshold+1]
	coefficients := polynomial.Lagrange(curve.Secp256k1{}, quorum)
	reconstructed := curve.Secp256k1{}.NewPoint()
	for _, id := range quorum {
		reconstructed = reconstructed.Add(coefficients[id].Act(config.VerificationShares.Points[id]))
	}
	if !reconstructed.Equal(publicKey) {
		return errors.New("FROST verification shares do not reconstruct the group public key")
	}

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
