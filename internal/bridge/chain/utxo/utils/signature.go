package utils

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/btcsuite/btcd/btcec/v2"
	ecdsabtc "github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

func EncodeSignature(sig tss.SignatureData, sigHashType byte) []byte {
	if sig == nil {
		return nil
	}

	r, s := new(btcec.ModNScalar), new(btcec.ModNScalar)
	r.SetByteSlice(sig.GetR())
	s.SetByteSlice(sig.GetS())

	// TODO: Maybe move it to signature level and add frost compatibility
	btcSig := ecdsabtc.NewSignature(r, s)

	return append(btcSig.Serialize(), sigHashType)
}
