package evm

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts"
	"github.com/pkg/errors"
)

const (
	EventDepositedNative = "DepositedNative"
	EventDepositedERC20  = "DepositedERC20"
)

var events = []string{
	EventDepositedNative,
	EventDepositedERC20,
}

type Client struct {
	chain         Chain
	contractABI   abi.ABI
	depositEvents []abi.Event
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain Chain) *Client {
	bridgeAbi, err := abi.JSON(strings.NewReader(contracts.BridgeMetaData.ABI))
	if err != nil {
		panic(errors.Wrap(err, "failed to parse bridge ABI"))
	}

	depositEvents := make([]abi.Event, len(events))
	for i, event := range events {
		depositEvent, ok := bridgeAbi.Events[event]
		if !ok {
			panic("wrong bridge ABI events")
		}
		depositEvents[i] = depositEvent
	}

	return &Client{
		chain:         chain,
		contractABI:   bridgeAbi,
		depositEvents: depositEvents,
	}
}

func (p *Client) ChainId() string {
	return p.chain.Id
}

func (p *Client) Type() chain.Type {
	return chain.TypeEVM
}

func (p *Client) getDepositLogType(log *types.Log) string {
	if log == nil || len(log.Topics) == 0 {
		return ""
	}

	for _, event := range p.depositEvents {
		isEqual := bytes.Equal(log.Topics[0].Bytes(), event.ID.Bytes())
		if isEqual {
			return event.Name
		}
	}

	return ""
}

func (p *Client) AddressValid(addr string) bool {
	return common.IsHexAddress(addr)
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}
