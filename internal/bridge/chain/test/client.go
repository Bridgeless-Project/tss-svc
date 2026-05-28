package test

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
)

type Client struct {
	Chain
	share tss.Share
	*DepositDecoder
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {

	return &Client{
		Chain:          chain,
		DepositDecoder: nil,
	}
}

func (c *Client) ChainId() string {
	return c.Id
}

func (c *Client) Share() tss.Share {
	return c.share
}

func (c *Client) SetShare(share tss.Share) {
	c.share = share
}

func (c *Client) Type() chain.Type {
	return chain.TypeOther
}

func (c *Client) AddressValid(_ string) bool {
	return true
}

func (c *Client) TransactionHashValid(_ string) bool {
	return true
}

func (c *Client) HealthCheck() error {
	return nil
}

func (c *Client) IsCentralized() bool {
	return false
}
