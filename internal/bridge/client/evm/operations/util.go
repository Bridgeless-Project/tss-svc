package operations

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

func ToBytes32(arr []byte) []byte {
	if len(arr) >= 32 {
		return arr[:32]
	}

	res := make([]byte, 32-len(arr))
	return append(res, arr...)
}

func IntToBytes32(amount int) []byte {
	return ToBytes32(big.NewInt(int64(amount)).Bytes())
}

func BoolToBytes(b bool) []byte {
	if b {
		return []byte{1}
	}

	return []byte{0}
}

func SetSignaturePrefix(message []byte) []byte {
	lenMessage := []byte(fmt.Sprintf("%d", len(message)))
	prefix := []byte("\x19Ethereum Signed Message:\n")
	prefixedMessage := bytes.Join([][]byte{prefix, lenMessage, message}, nil)

	return crypto.Keccak256(prefixedMessage)
}
