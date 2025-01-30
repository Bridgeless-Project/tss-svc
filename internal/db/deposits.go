package db

import (
	"fmt"
	"math/big"

	"github.com/hyle-team/tss-svc/internal/types"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

const OriginTxIdPattern = "%s-%d-%s"

var ErrAlreadySubmitted = errors.New("transaction already submitted")
var FinalWithdrawalStatuses = []types.WithdrawalStatus{
	//transaction is signed
	types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
	// data invalid or something goes wrong
	types.WithdrawalStatus_WITHDRAWAL_STATUS_INVALID,
	types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED,
}

type DepositsQ interface {
	New() DepositsQ
	Insert(Deposit) (id int64, err error)
	Select(selector DepositsSelector) ([]Deposit, error)
	Get(identifier DepositIdentifier) (*Deposit, error)
	GetWithSelector(selector DepositsSelector) (*Deposit, error)

	UpdateWithdrawalDetails(DepositIdentifier, string, string) error
	UpdateSignature(DepositIdentifier, string) error
	UpdateStatus(DepositIdentifier, types.WithdrawalStatus) error

	Transaction(f func() error) error
}

type WithdrawalTx struct {
	DepositId int64
	TxHash    string
	ChainId   string
}

type DepositIdentifier struct {
	TxHash  string `structs:"tx_hash" db:"tx_hash"`
	TxNonce int    `structs:"tx_nonce" db:"tx_nonce"`
	ChainId string `structs:"chain_id" db:"chain_id"`
}

type DepositsSelector struct {
	Ids     []int64
	ChainId *string
	One     bool
	Status  *types.WithdrawalStatus
}

func (d DepositIdentifier) String() string {
	return fmt.Sprintf(OriginTxIdPattern, d.TxHash, d.TxNonce, d.ChainId)
}

type Deposit struct {
	Id int64 `structs:"-" db:"id"`
	DepositIdentifier

	Depositor       *string `structs:"depositor" db:"depositor"`
	DepositAmount   string  `structs:"deposit_amount" db:"deposit_amount"`
	DepositToken    string  `structs:"deposit_token" db:"deposit_token"`
	Receiver        string  `structs:"receiver" db:"receiver"`
	WithdrawalToken string  `structs:"withdrawal_token" db:"withdrawal_token"`
	DepositBlock    int64   `structs:"deposit_block" db:"deposit_block"`

	WithdrawalStatus types.WithdrawalStatus `structs:"withdrawal_status" db:"withdrawal_status"`

	WithdrawalTxHash  *string `structs:"withdrawal_tx_hash" db:"withdrawal_tx_hash"`
	WithdrawalChainId string  `structs:"withdrawal_chain_id" db:"withdrawal_chain_id"`
	WithdrawalAmount  string  `structs:"withdrawal_amount" db:"withdrawal_amount"`

	IsWrappedToken bool `structs:"is_wrapped_token" db:"is_wrapped_token"`

	Signature *string `structs:"signature" db:"signature"`
}

type DepositData struct {
	DepositIdentifier

	Block         int64
	SourceAddress string
	DepositAmount *big.Int
	TokenAddress  string

	DestinationAddress string
	DestinationChainId string
}

func (d DepositData) ToNewDeposit(
	withdrawalAmount *big.Int,
	dstTokenAddress string,
	isWrappedToken bool,
) Deposit {
	return Deposit{
		DepositIdentifier: d.DepositIdentifier,
		Depositor:         &d.SourceAddress,
		DepositAmount:     d.DepositAmount.String(),
		DepositToken:      d.TokenAddress,
		Receiver:          d.DestinationAddress,
		WithdrawalToken:   dstTokenAddress,
		DepositBlock:      d.Block,
		WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING,
		WithdrawalChainId: d.DestinationChainId,
		WithdrawalAmount:  withdrawalAmount.String(),
		IsWrappedToken:    isWrappedToken,
	}
}

func (d DepositData) OriginTxId() string {
	return d.DepositIdentifier.String()
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
