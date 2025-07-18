package bridge

import (
	"math/big"
	"regexp"
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
