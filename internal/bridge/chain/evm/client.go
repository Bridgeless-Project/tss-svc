package evm

import (
	"context"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	v1 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v1"
	v2 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

var requiredEvents = []string{
	EventNameDepositedNative,
	EventNameDepositedERC20,
}

type Client struct {
	chain           Chain
	abiV1           abi.ABI
	abiV2           abi.ABI
	supportedEvents map[string]EventType
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {
	abiV1, err := abi.JSON(strings.NewReader(v1.BridgeMetaData.ABI))
	if err != nil {
		panic(errors.Wrap(err, "failed to parse bridge ABI v1"))
	}
	abiV2, err := abi.JSON(strings.NewReader(v2.BridgeMetaData.ABI))
	if err != nil {
		panic(errors.Wrap(err, "failed to parse bridge ABI v2"))
	}

	supportedEvents := make(map[string]EventType)
	for _, eventName := range requiredEvents {
		v1Event, ok := abiV1.Events[eventName]
		if !ok {
			panic("required eventName not found in ABI v1: " + eventName)
		}
		supportedEvents[v1Event.ID.String()] = EventNameToEventV1[eventName]

		v2Event, ok := abiV2.Events[eventName]
		if !ok {
			panic("required eventName not found in ABI v2: " + eventName)
		}
		supportedEvents[v2Event.ID.String()] = EventNameToEventV2[eventName]
	}

	return &Client{
		chain:           chain,
		abiV1:           abiV1,
		abiV2:           abiV2,
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
	if _, err := p.chain.Rpc.BlockNumber(context.Background()); err != nil {
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
