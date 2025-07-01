package ton

import (
	"github.com/xssnick/tonutils-go/address"
	"math/big"
)

const (
	depositNativeOpCode        = "0xe858a993"
	depositJettonOpCode        = "0x02ddcbe3"
	opCodeBitSize              = 32
	intBitSize                 = 257
	receiverBitSize            = 256
	withdrawalNativeHashMethod = "nativeHash"
	withdrawalJettonHashMethod = "jettonHash"
	trueBit                    = -1
)

type depositJettonContent struct {
	Sender       *address.Address
	Amount       *big.Int
	Receiver     string
	ChainId      string
	IsWrapped    bool
	TokenAddress *address.Address
}

type depositNativeContent struct {
	Sender   *address.Address
	Amount   *big.Int
	Receiver string
	ChainId  string
}
