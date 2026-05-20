package tss

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/pkg/errors"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

func Verify(pk *ecdsa.PublicKey, inputData []byte, signature *common.SignatureData) bool {
	if pk == nil || signature == nil {
		return false
	}

	data := big.NewInt(0).SetBytes(inputData)
	r, s := new(big.Int).SetBytes(signature.R), new(big.Int).SetBytes(signature.S)

	return ecdsa.Verify(pk, data.Bytes(), r, s)
}

func FrostPubKey(share *frostkeygen.Config) ([]byte, error) {
	if share == nil {
		return nil, errors.New("nil FROST share")
	}

	publicKey, ok := share.PublicKey.(*curve.Secp256k1Point)
	if !ok {
		return nil, errors.New("FROST public key is not secp256k1")
	}

	return publicKey.XBytes(), nil
}

func VerifyFrost(pubKey []byte, inputData []byte, signature *common.SignatureData) bool {
	if len(pubKey) == 0 || signature == nil {
		return false
	}

	// TODO: do not use taproot for ZCash
	if len(signature.Signature) != taproot.SignatureLen {
		return false
	}

	return taproot.PublicKey(pubKey).Verify(signature.Signature, inputData)
}
