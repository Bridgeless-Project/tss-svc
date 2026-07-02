package zano

import (
	"errors"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Zano supports only ECDSA
func EncodeSignature(signature tss.SignatureData) (string, error) {
	rawSig, err := tss.RecoverableECDSASignatureBytes(signature)
	if err != nil {
		return "", err
	}
	encoded := hexutil.Encode(rawSig)
	if len(encoded) < 4 {
		return "", errors.New("encoded signature is too short")
	}

	// stripping redundant hex-prefix and recovery byte (two hex-characters)
	strippedSignature := encoded[2 : len(encoded)-2]

	return strippedSignature, nil
}
