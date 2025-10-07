package ton

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
)

type Client struct {
	Chain
	*DepositDecoder
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
	api.WithTimeout(chain.RPC.Timeout * time.Second)

	chain.Client = api

	depositDecoder := NewDepositDecoder(*chain.BridgeContractAddress, chain.RPC.IsTestnet)

	return &Client{
		chain,
		depositDecoder,
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

func (c *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Client.GetMasterchainInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get masterchain info from ton client")
	}

	return nil
}
