package bitcoin

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
)

const ConsolidationThreshold = 20

var dustAmount = big.NewInt(547)

type Client struct {
	chain     Chain
	mockedKey *btcec.PrivateKey
}

func NewBridgeClient(chain Chain) *Client {
	mockedKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to create mocked private key: %v", err))
	}

	return &Client{chain, mockedKey}
}

func (c *Client) ConsolidationThreshold() int {
	return ConsolidationThreshold
}

func (c *Client) ChainId() string {
	return c.chain.Id
}

func (c *Client) Type() chain.Type {
	return chain.TypeBitcoin
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

func (c *Client) ChainParams() *chaincfg.Params {
	return c.chain.Params
}

func (c *Client) IsBridgeAddr(addr btcutil.Address) bool {
	for _, receiver := range c.chain.Receivers {
		if addr.String() == receiver.String() {
			return true
		}
	}

	return false
}
