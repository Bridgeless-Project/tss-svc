package zano

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func EncodeSignature(signature tss.SignatureData) string {
	if signature == nil {
		return ""
	}

	rawSig := append(signature.GetSignature(), signature.GetSignatureRecovery()...)
	encoded := hexutil.Encode(rawSig)

	// stripping redundant hex-prefix and recovery byte (two hex-characters)
	strippedSignature := encoded[2 : len(encoded)-2]

	return strippedSignature
}
