package utxo

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/factory"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func PubkeyToAddress(x, y *big.Int, chain types.Chain, network types.Network) string {
	hlp := factory.NewUtxoHelper(chain, network)
	pubkey := &ecdsa.PublicKey{Curve: crypto.S256(), X: x, Y: y}

	return hlp.P2pkhAddress(pubkey)
}
