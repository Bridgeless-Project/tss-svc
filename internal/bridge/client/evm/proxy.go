package evm

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm/contracts"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

const (
	EventDepositedNative = "DepositedNative"
	EventDepositedERC20  = "DepositedERC20"
)

var events = []string{
	EventDepositedNative,
	EventDepositedERC20,
}

type BridgeClient interface {
	bridgeTypes.Client
	GetSignHash(deposit db.Deposit) ([]byte, error)
}

type client struct {
	chain         chain.EvmChain
	contractABI   abi.ABI
	depositEvents []abi.Event
	logger        *logan.Entry
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain chain.EvmChain) BridgeClient {
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

	return &client{
		chain:         chain,
		contractABI:   bridgeAbi,
		depositEvents: depositEvents,
	}
}

func (p *client) ChainId() string {
	return p.chain.Id
}

func (p *client) Type() chain.Type {
	return chain.TypeEVM
}

func (p *client) getDepositLogType(log *types.Log) string {
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

func (p *client) AddressValid(addr string) bool {
	return common.IsHexAddress(addr)
}

func (p *client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}
