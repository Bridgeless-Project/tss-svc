package chain

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
)

var (
	ErrChainNotSupported      = errors.New("chain not supported")
	ErrTxPending              = errors.New("transaction is pending")
	ErrTxFailed               = errors.New("transaction failed")
	ErrTxNotFound             = errors.New("transaction not found")
	ErrDepositNotFound        = errors.New("deposit not found")
	ErrTxNotConfirmed         = errors.New("transaction not confirmed")
	ErrInvalidReceiverAddress = errors.New("invalid receiver address")
	ErrInvalidBridgeId        = errors.New("invalid bridge id")
	ErrInvalidDepositedAmount = errors.New("invalid deposited amount")
	ErrInvalidScriptPubKey    = errors.New("invalid script pub key")
	ErrInvalidTxNonce         = errors.New("invalid tx nonce")
	ErrFailedUnpackLogs       = errors.New("failed to unpack logs")
	ErrUnsupportedEvent       = errors.New("unsupported event")
	ErrUnsupportedContract    = errors.New("unsupported contract")
	ErrInvalidTransactionData = errors.New("invalid transaction data")
)

func IsPendingDepositError(err error) bool {
	return errors.Is(err, ErrTxPending) ||
		errors.Is(err, ErrTxNotConfirmed)
}

func IsInvalidDepositError(err error) bool {
	return errors.Is(err, ErrChainNotSupported) ||
		errors.Is(err, ErrTxFailed) ||
		errors.Is(err, ErrTxNotFound) ||
		errors.Is(err, ErrDepositNotFound) ||
		errors.Is(err, ErrInvalidReceiverAddress) ||
		errors.Is(err, ErrInvalidDepositedAmount) ||
		errors.Is(err, ErrInvalidScriptPubKey) ||
		errors.Is(err, ErrInvalidTxNonce) ||
		errors.Is(err, ErrFailedUnpackLogs) ||
		errors.Is(err, ErrUnsupportedEvent) ||
		errors.Is(err, ErrUnsupportedContract)
}

type Client interface {
	Type() Type
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

type Chain struct {
	Id              string `fig:"id,required"`
	Type            Type   `fig:"type,required"`
	Confirmations   uint64 `fig:"confirmations,required"`
	Rpc             any    `fig:"rpc,required"`
	BridgeAddresses any    `fig:"bridge_addresses,required"`

	Meta any `fig:"meta"`
}

type Type string

const (
	TypeEVM     Type = "evm"
	TypeZano    Type = "zano"
	TypeBitcoin Type = "bitcoin"
	TypeTON     Type = "ton"
	TypeSolana  Type = "solana"
	TypeOther   Type = "other"
)

var typesMap = map[Type]struct{}{
	TypeEVM:     {},
	TypeZano:    {},
	TypeOther:   {},
	TypeBitcoin: {},
	TypeTON:     {},
	TypeSolana:  {},
}

func (c Type) Validate() error {
	if _, ok := typesMap[c]; !ok {
		return errors.New("invalid chain type")
	}

	return nil
}
