package test

import (
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/xssnick/tonutils-go/address"
)

// MOCK
func (c *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	data := new(db.DepositData)
	data.TxHash = id.TxHash
	data.DepositIdentifier = id
	data.ChainId = "test"
	data.DestinationAddress = "0x9F2C0E3DeE0B50ba9e97A9e88a2f564Cc43B5627"
	data.DestinationChainId = "2" // test chain id
	data.TokenAddress = "0x0000000000000000000000000000000000000000"

	return data, nil
}

// DepositDecoder is a struct that decodes deposit messages from the TON blockchain.
// It implements all the methods required to parse deposit messages and extract relevant data.
type DepositDecoder struct {
	bridgeAddress address.Address
}

func NewDepositDecoder(bridgeAddress address.Address, _ bool) *DepositDecoder {
	return &DepositDecoder{
		bridgeAddress: bridgeAddress,
	}
}
