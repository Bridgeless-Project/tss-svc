package types

import (
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"math/big"
)

var (
	ErrChainNotSupported      = errors.New("chain not supported")
	ErrInvalidProxyType       = errors.New("invalid proxy type")
	ErrTxPending              = errors.New("transaction is pending")
	ErrTxFailed               = errors.New("transaction failed")
	ErrTxNotFound             = errors.New("transaction not found")
	ErrDepositNotFound        = errors.New("deposit not found")
	ErrTxNotConfirmed         = errors.New("transaction not confirmed")
	ErrInvalidReceiverAddress = errors.New("invalid receiver address")
	ErrInvalidDepositedAmount = errors.New("invalid deposited amount")
	ErrInvalidScriptPubKey    = errors.New("invalid script pub key")
	ErrFailedUnpackLogs       = errors.New("failed to unpack logs")
	ErrUnsupportedEvent       = errors.New("unsupported event")
	ErrUnsupportedContract    = errors.New("unsupported contract")
)

type ChainType string

const (
	ChainTypeEVM     ChainType = "evm"
	ChainTypeBitcoin ChainType = "bitcoin"
	ChainTypeZano    ChainType = "zano"
	ChainTypeOther   ChainType = "other"
)

func (c ChainType) Validate() error {
	switch c {
	case ChainTypeEVM, ChainTypeBitcoin, ChainTypeZano, ChainTypeOther:
		return nil
	default:
		return errors.New("invalid chain type")
	}
}

type TransactionStatus int8

const (
	TransactionStatusPending TransactionStatus = iota
	TransactionStatusSuccessful
	TransactionStatusFailed
	TransactionStatusNotFound
	TransactionStatusUnknown
)

type Proxy interface {
	Type() ChainType
	GetTransactionStatus(txHash string) (TransactionStatus, error)
	GetDepositData(id db.DepositIdentifier) (*db.DepositData, error)

	AddressValid(addr string) bool
	TransactionHashValid(hash string) bool
	WithdrawalAmountValid(amount *big.Int) bool
	// TODO: add withdrawal transaction constructor and data validator
}

type ProxiesRepository interface {
	Proxy(chainId string) (Proxy, error)
	SupportsChain(chainId string) bool
}
