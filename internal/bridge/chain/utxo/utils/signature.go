package utils

import (
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/btcec/v2"
	ecdsabtc "github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

func EncodeSignature(sig *common.SignatureData, sigHashType byte) []byte {
	if sig == nil {
		return nil
	}

	r, s := new(btcec.ModNScalar), new(btcec.ModNScalar)
	r.SetByteSlice(sig.R)
	s.SetByteSlice(sig.S)

	btcSig := ecdsabtc.NewSignature(r, s)

	return append(btcSig.Serialize(), sigHashType)
}
