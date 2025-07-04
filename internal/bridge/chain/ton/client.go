package ton

import (
	"context"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
)

type Client struct {
	Chain
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {
	liteClt := liteclient.NewConnectionPool()
	err := liteClt.AddConnectionsFromConfigUrl(context.Background(), chain.RPC.GlobalConfigUrl)
	if err != nil {
		panic(errors.Wrap(err, "failed to connect to global config"))
	}
	globalConfig, err := liteclient.GetConfigFromUrl(context.Background(), chain.RPC.GlobalConfigUrl)

	api := ton.NewAPIClient(liteClt, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(globalConfig)
	api.WithTimeout(time.Duration(chain.RPC.Timeout) * time.Second)

	chain.Client = api

	return &Client{
		chain,
	}
}

func (c *Client) ChainId() string {
	return c.Id
}

func (c *Client) Type() chain.Type {
	return chain.TypeTON
}

func (c *Client) AddressValid(addr string) bool {
	_, err := address.ParseAddr(addr)
	return err == nil
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}
