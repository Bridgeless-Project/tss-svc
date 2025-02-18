package bitcoin

import (
	"math/big"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chains"
)

var dustAmount = big.NewInt(547)

type Client struct {
	chain chains.Bitcoin
}

func NewBridgeClient(chain chains.Bitcoin) *Client {
	return &Client{chain}
}

func (c *Client) ChainId() string {
	return c.chain.Id
}

func (c *Client) Type() chains.Type {
	return chains.TypeBitcoin
}

func (c *Client) AddressValid(addr string) bool {
	_, err := btcutil.DecodeAddress(addr, c.chain.Params)
	return err == nil
}

func (c *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func (c *Client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(dustAmount) == -1 {
		return false
	}

	return true
}
