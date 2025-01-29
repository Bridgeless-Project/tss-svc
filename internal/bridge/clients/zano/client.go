package zano

import (
	"regexp"

	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chains"
)

var addressPattern = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{97}$`)

type Client struct {
	chain chains.Zano
}

func (p *Client) ChainId() string {
	return p.chain.Id
}

func (p *Client) Type() chains.Type {
	return chains.TypeZano
}

func (p *Client) AddressValid(addr string) bool {
	return addressPattern.MatchString(addr)
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func NewBridgeClient(chain chains.Zano) *Client {
	return &Client{chain}
}
