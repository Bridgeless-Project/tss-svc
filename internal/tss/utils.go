package tss

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
)

func Verify(pk *ecdsa.PublicKey, inputData []byte, signature *common.SignatureData) bool {
	data := big.NewInt(0).SetBytes(inputData)
	r, s := new(big.Int).SetBytes(signature.R), new(big.Int).SetBytes(signature.S)

	return ecdsa.Verify(pk, data.Bytes(), r, s)
}
