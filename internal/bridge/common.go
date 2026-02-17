package bridge

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"
	"regexp"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

const (
	HexPrefix                 = "0x"
	DefaultNativeTokenAddress = "0x0000000000000000000000000000000000000000"
)

var (
	ZeroAmount                    = big.NewInt(0)
	DefaultTransactionHashPattern = regexp.MustCompile("^0x[a-fA-F0-9]{64}$")
	SolanaTransactionHashPattern  = regexp.MustCompile("^[1-9A-HJ-NP-Za-km-z]{86,88}$")
)

func PubkeyToString(x, y *big.Int) string {
	marshaled := elliptic.Marshal(tss.S256(), x, y)

	// Marshaled point contains constant 0x04 first byte, we have to remove it
	return hexutil.Encode(marshaled[1:])
}

func PubkeyCompressedToString(x, y *big.Int) string {
	marshalled := elliptic.Marshal(tss.S256(), x, y)

	key, err := crypto.UnmarshalPubkey(marshalled)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal pubkey: %v", err))
	}

	return hexutil.Encode(crypto.CompressPubkey(key))
}

func DecodePubkey(pubkeyStr string) (*ecdsa.PublicKey, error) {
	pubkeyBytes, err := hexutil.Decode(pubkeyStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode pubkey")
	}

	// crypto.UnmarshalPubkey expects uncompressed pubkey with 0x04 prefix
	if pubkeyBytes[0] != 0x04 {
		pubkeyBytes = append([]byte{0x04}, pubkeyBytes...)
	}

	pubkey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal pubkey")
	}

	return pubkey, nil
}
