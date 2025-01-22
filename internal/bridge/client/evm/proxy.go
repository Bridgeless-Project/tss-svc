package evm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/contracts"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"strings"
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
	GetSignHash(data db.DepositData) ([]byte, error)
}

type client struct {
	chain         chain.EvmChain
	contractABI   abi.ABI
	depositEvents []abi.Event
	logger        *logan.Entry
}

func (p *client) ConstructWithdrawalTx(data db.Deposit) ([]byte, error) {
	withdrawalTx := db.WithdrawalTx{
		DepositId: data.Id,
		TxHash:    data.TxHash,
		ChainId:   *data.WithdrawalChainId,
	}

	dataToSign, err := json.Marshal(withdrawalTx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct data")
	}
	dataHash := crypto.Keccak256Hash(dataToSign)
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(dataHash.Bytes()), dataHash.Bytes())
	return crypto.Keccak256Hash([]byte(msg)).Bytes(), nil
}

// NewBridgeClient creates a new bridge Client for the given chain.
func NewBridgeClient(chain chain.EvmChain, logger *logan.Entry) BridgeClient {
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
		logger:        logger,
	}
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
