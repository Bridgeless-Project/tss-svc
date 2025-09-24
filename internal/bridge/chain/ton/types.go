package ton

import (
	"math/big"

	"github.com/xssnick/tonutils-go/address"
)

const (
	depositNativeOpCode        = "0xe858a993"
	depositJettonOpCode        = "0x02ddcbe3"
	opCodeBitSize              = 32
	networkCellSizeBytes       = 32
	networkCellSizeBit         = 256
	amountBitSize              = 257
	receiverBitSize            = 1016
	referralBitSize            = 16
	withdrawalNativeHashMethod = "nativeHash"
	withdrawalJettonHashMethod = "jettonHash"
	trueBit                    = -1

	receiverCellId = 0
	networkCellId  = 1
)

type depositJettonContent struct {
	Sender       *address.Address
	Amount       *big.Int
	Receiver     string
	ChainId      string
	IsWrapped    bool
	TokenAddress *address.Address
	ReferralId   uint16
}

type depositNativeContent struct {
	Sender     *address.Address
	Amount     *big.Int
	Receiver   string
	ChainId    string
	ReferralId uint16
}
