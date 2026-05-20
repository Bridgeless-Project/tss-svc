package evm

import (
	"crypto/elliptic"
	"math/big"

	tssCommon "github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func PubkeyToAddress(x, y *big.Int) common.Address {
	marshalled := elliptic.Marshal(tss.S256(), x, y)
	// Marshalled point contains constant 0x04 first byte, we do not have to include it
	hash := crypto.Keccak256(marshalled[1:])

	// The Ethereum address is the last 20 bytes of the hash (hash[12:32])
	return common.BytesToAddress(hash[12:])
}

func ConvertSignature(sig *tssCommon.SignatureData) string {
	rawSig := append(sig.Signature, sig.SignatureRecovery...)
	rawSig[64] += 27

	return hexutil.Encode(rawSig)
}
