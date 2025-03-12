package zano

import (
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func EncodeSignature(signature *common.SignatureData) string {
	if signature == nil {
		return ""
	}

	rawSig := append(signature.Signature, signature.SignatureRecovery...)
	encoded := hexutil.Encode(rawSig)

	// stripping redundant hex-prefix and recovery byte (two hex-characters)
	strippedSignature := encoded[2 : len(encoded)-2]

	return strippedSignature
}
