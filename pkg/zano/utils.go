package zano

import "github.com/ethereum/go-ethereum/common/hexutil"

const hexPrefix = "0x"

// FormSigningData creates a signing data required for externally signed transactions.
func FormSigningData(txId string) []byte {
	return hexutil.MustDecode(hexPrefix + txId)
}
