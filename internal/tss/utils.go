package tss

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func Verify(localParty *keygen.LocalPartySaveData, inputData []byte, signature *common.SignatureData) bool {
	pk := ecdsa.PublicKey{
		Curve: tss.EC(),
		X:     localParty.ECDSAPub.X(),
		Y:     localParty.ECDSAPub.Y(),
	}
	data := big.NewInt(0).SetBytes(inputData)

	return ecdsa.Verify(&pk, data.Bytes(), new(big.Int).SetBytes(signature.R), new(big.Int).SetBytes(signature.S))
}
