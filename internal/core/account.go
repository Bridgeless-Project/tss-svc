package core

import (
	"github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	secp256k1 "github.com/hyle-team/bridgeless-core/v12/crypto/ethsecp256k1"
	"github.com/pkg/errors"
)

const defaultHrp = "bridge"

type Account struct {
	privKey *secp256k1.PrivKey
	addr    Address
}

func NewAccount(privKey string, hrp ...string) (*Account, error) {
	key := &secp256k1.PrivKey{Key: hexutil.MustDecode(privKey)}

	prefix := defaultHrp
	if len(hrp) > 0 {
		prefix = hrp[0]
	}

	address, err := bech32.ConvertAndEncode(prefix, key.PubKey().Address().Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert and encode address")
	}

	return &Account{
		privKey: key,
		addr:    Address(address),
	}, nil
}

func (a *Account) PrivateKey() *secp256k1.PrivKey {
	return a.privKey
}

func (a *Account) PublicKey() types.PubKey {
	return a.privKey.PubKey()
}

func (a *Account) CosmosAddress() Address {
	return a.addr
}
