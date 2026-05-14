package evm

import (
	"fmt"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

type Client struct {
	chain           Chain
	abis            map[string]abi.ABI
	supportedEvents map[string]EventType

	reqGroup singleflight.Group
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {
	versions := supportedVersions

	versionedABIs := make(map[string]abi.ABI)
	supportedEvents := make(map[string]EventType)

	for vName, ver := range versions {
		parsed, err := abi.JSON(strings.NewReader(ver.metadata.ABI))
		if err != nil {
			panic(fmt.Sprintf("failed to parse ABI for %s: %v", vName, err))
		}

		versionedABIs[vName] = parsed

		for eventName, eventType := range ver.eventNameToType {
			event, ok := parsed.Events[eventName]
			if !ok {
				panic(fmt.Sprintf("required event %s not found in ABI version %s", eventName, vName))
			}

			supportedEvents[event.ID.String()] = eventType
		}
	}

	return &Client{
		chain:           chain,
		abis:            versionedABIs,
		supportedEvents: supportedEvents,
	}
}

func (p *Client) ChainId() string {
	return p.chain.Id
}

func (p *Client) Type() chain.Type {
	return chain.TypeEVM
}

func (p *Client) AddressValid(addr string) bool {
	return common.IsHexAddress(addr)
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func (p *Client) HealthCheck() error {
	if _, err := p.GetCurrentBlockNumber(); err != nil {
		return errors.Wrap(err, "failed to check block number")
	}

	return nil
}

func (p *Client) GetDepositEventType(log *types.Log) EventType {
	if log == nil || len(log.Topics) == 0 {
		return ""
	}

	return p.supportedEvents[log.Topics[0].String()]
}

func (p *Client) IsCentralized() bool {
	return p.chain.Meta.Centralized
}

func (p *Client) IsStandart() bool {
	return p.chain.Meta.Standart
}
