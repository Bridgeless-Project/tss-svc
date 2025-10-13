package zano

import (
	"regexp"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
)

var addressPattern = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{97}$`)

type Client struct {
	chain Chain
}

func (p *Client) ChainId() string {
	return p.chain.Id
}

func (p *Client) Type() chain.Type {
	return chain.TypeZano
}

func (p *Client) AddressValid(addr string) bool {
	return addressPattern.MatchString(addr)
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func NewBridgeClient(chain Chain) *Client {
	return &Client{chain}
}

func (p *Client) HealthCheck() error {
	_, err := p.chain.Client.CurrentHeight()
	if err != nil {
		return errors.Wrap(err, "failed to get current height from zano daemon")
	}

	_, err = p.chain.Client.GetWalletInfo()
	if err != nil {
		return errors.Wrap(err, "failed to get wallet info from zano wallet")
	}

	return nil
}
