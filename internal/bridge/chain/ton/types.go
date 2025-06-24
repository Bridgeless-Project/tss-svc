package ton

import (
	"math/big"
)

const (
	depositNativeOpCode = "0xe858a993"
	depositJettonOpCode = "0x02ddcbe3"
	decimals            = 9
	opCodeBitSize       = 32
	intBitSize          = 257
)

type depositJettonContent struct {
	Sender       string
	Amount       *big.Int
	Receiver     string
	Network      string
	IsWrapped    bool
	TokenAddress string
}

type depositNativeContent struct {
	Sender   string
	Amount   *big.Int
	Receiver string
	Network  string
}
