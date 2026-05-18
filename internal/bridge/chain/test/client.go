package test

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
)

type Client struct {
	Chain
	*DepositDecoder
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {

	return &Client{
		chain,
		nil,
	}
}

func (c *Client) ChainId() string {
	return c.Id
}

func (c *Client) Type() chain.Type {
	return chain.TypeOther
}

func (c *Client) AddressValid(addr string) bool {
	return true
}

func (c *Client) TransactionHashValid(hash string) bool {
	return true
}

func (c *Client) HealthCheck() error {
	return nil
}

func (c *Client) IsCentralized() bool {
	return false
}
