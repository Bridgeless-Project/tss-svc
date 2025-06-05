package ton

import (
	"context"
	"encoding/hex"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"log"
	"time"

	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

// TODO: Add contract events map

type Client struct {
	Chain
	logger *logan.Entry
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {
	liteClt := liteclient.NewConnectionPool()
	err := liteClt.AddConnectionsFromConfigUrl(context.Background(), chain.RPC.GlobalConfigUrl)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to connect to global config"))
	}
	globalConfig, err := liteclient.GetConfigFromUrl(context.Background(), chain.RPC.GlobalConfigUrl)

	api := ton.NewAPIClient(liteClt, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(globalConfig)
	api.WithTimeout(time.Duration(chain.RPC.Timeout) * time.Second)

	chain.Client = api

	return &Client{
		chain, nil,
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
	_, err := hex.DecodeString(hash)
	return err == nil
}
