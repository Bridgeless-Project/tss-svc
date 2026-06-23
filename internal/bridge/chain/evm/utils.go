package evm

import (
	"crypto/elliptic"
	"math/big"

	tss2 "github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v3/tss"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

func PubkeyToAddress(x, y *big.Int) common.Address {
	marshalled := elliptic.Marshal(tss.S256(), x, y)
	// Marshalled point contains constant 0x04 first byte, we do not have to include it
	hash := crypto.Keccak256(marshalled[1:])

	// The Ethereum address is the last 20 bytes of the hash (hash[12:32])
	return common.BytesToAddress(hash[12:])
}

func ConvertSignature(sig tss2.SignatureData) (string, error) {
	if sig == nil {
		return "", errors.New("nil signature")
	}

	r := sig.GetR()
	if len(r) != 32 {
		return "", errors.Errorf("invalid EVM signature R length: %d", len(r))
	}
	s := sig.GetS()
	if len(s) != 32 {
		return "", errors.Errorf("invalid EVM signature S length: %d", len(s))
	}
	recovery := sig.GetSignatureRecovery()
	if len(recovery) != 1 {
		return "", errors.Errorf("invalid EVM signature recovery length: %d", len(recovery))
	}

	v := recovery[0]
	switch v {
	case 0, 1:
		v += 27
	case 27, 28:
	default:
		return "", errors.Errorf("invalid EVM signature recovery value: %d", v)
	}

	rawSig := make([]byte, 0, 65)
	rawSig = append(rawSig, r...)
	rawSig = append(rawSig, s...)
	rawSig = append(rawSig, v)

	return hexutil.Encode(rawSig), nil
}
