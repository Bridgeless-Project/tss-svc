package clients

import (
	"math/big"

	"github.com/hyle-team/tss-svc/internal/bridge/chains"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

var (
	ErrChainNotSupported      = errors.New("chains not supported")
	ErrTxPending              = errors.New("transaction is pending")
	ErrTxFailed               = errors.New("transaction failed")
	ErrTxNotFound             = errors.New("transaction not found")
	ErrDepositNotFound        = errors.New("withdrawal not found")
	ErrTxNotConfirmed         = errors.New("transaction not confirmed")
	ErrInvalidReceiverAddress = errors.New("invalid receiver address")
	ErrInvalidDepositedAmount = errors.New("invalid deposited amount")
	ErrInvalidScriptPubKey    = errors.New("invalid script pub key")
	ErrFailedUnpackLogs       = errors.New("failed to unpack logs")
	ErrUnsupportedEvent       = errors.New("unsupported event")
	ErrUnsupportedContract    = errors.New("unsupported contract")
)

func IsPendingDepositError(err error) bool {
	return errors.Is(err, ErrTxPending) ||
		errors.Is(err, ErrTxNotFound) ||
		errors.Is(err, ErrTxNotConfirmed)
}

func IsInvalidDepositError(err error) bool {
	return errors.Is(err, ErrChainNotSupported) ||
		errors.Is(err, ErrTxFailed) ||
		errors.Is(err, ErrDepositNotFound) ||
		errors.Is(err, ErrInvalidReceiverAddress) ||
		errors.Is(err, ErrInvalidDepositedAmount) ||
		errors.Is(err, ErrInvalidScriptPubKey) ||
		errors.Is(err, ErrFailedUnpackLogs) ||
		errors.Is(err, ErrUnsupportedEvent) ||
		errors.Is(err, ErrUnsupportedContract)
}

type Client interface {
	Type() chains.Type
	ChainId() string
	GetDepositData(id db.DepositIdentifier) (*db.DepositData, error)

	AddressValid(addr string) bool
	TransactionHashValid(hash string) bool
	WithdrawalAmountValid(amount *big.Int) bool
}

type Repository interface {
	Client(chainId string) (Client, error)
	SupportsChain(chainId string) bool
}
