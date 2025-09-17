package solana

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/gagliardetto/solana-go"
	"github.com/pkg/errors"
)

type Client struct {
	chain Chain
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {
	return &Client{
		chain: chain,
	}
}

func (p *Client) ChainId() string {
	return p.chain.Id
}

func (p *Client) Type() chain.Type {
	return chain.TypeSolana
}

func (p *Client) AddressValid(addr string) bool {
	_, err := solana.PublicKeyFromBase58(addr)
	return err == nil
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.SolanaTransactionHashPattern.MatchString(hash)
}

func (p *Client) HealthCheck() error {
	status, err := p.chain.Rpc.GetHealth(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to check node health")
	}
	if status != "ok" {
		return errors.New("node is not healthy with status: " + status)
	}

	return nil
}
