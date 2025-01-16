package db

import (
	"fmt"
	bridgetypes "github.com/hyle-team/bridgeless-core/x/bridge/types"
	"github.com/hyle-team/tss-svc/resources"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"math/big"
)

const OriginTxIdPattern = "%s-%d-%s"

var ErrAlreadySubmitted = errors.New("transaction already submitted")
var FinalWithdrawalStatuses = []resources.WithdrawalStatus{
	// transaction is sent
	resources.WithdrawalStatus_TX_PENDING,
	resources.WithdrawalStatus_TX_SUCCESSFUL,
	resources.WithdrawalStatus_TX_FAILED,
	// ready to be sent
	resources.WithdrawalStatus_WITHDRAWAL_SIGNED,
	// data invalid or something goes wrong
	resources.WithdrawalStatus_INVALID,
	resources.WithdrawalStatus_FAILED,
}

type DepositsQ interface {
	New() DepositsQ
	Insert(Deposit) (id int64, err error)
	Select(selector DepositsSelector) ([]Deposit, error)
	Get(identifier DepositIdentifier) (*Deposit, error)
	SetDepositData(data DepositData) error
	UpdateWithdrawalStatus(status resources.WithdrawalStatus, ids ...int64) error
	UpdateSubmitStatus(status resources.SubmitWithdrawalStatus, ids ...int64) error
	SetWithdrawalTxs(txs ...WithdrawalTx) error
	Transaction(f func() error) error
	SetDepositSignature(data DepositData) error
}

type WithdrawalTx struct {
	DepositId int64
	TxHash    string
	ChainId   string
}

type DepositIdentifier struct {
	TxHash    string `structs:"tx_hash" db:"tx_hash"`
	TxEventId int    `structs:"tx_event_id" db:"tx_event_id"`
	ChainId   string `structs:"chain_id" db:"chain_id"`
}

type DepositsSelector struct {
	Ids       []int64
	Submitted *bool
}

func (d DepositIdentifier) String() string {
	return fmt.Sprintf(OriginTxIdPattern, d.TxHash, d.TxEventId, d.ChainId)
}

type Deposit struct {
	Id int64 `structs:"-" db:"id"`
	DepositIdentifier
	Status resources.WithdrawalStatus `structs:"status" db:"status"`

	Depositor       *string `structs:"depositor" db:"depositor"`
	DepositAmount   *string `structs:"deposit_amount" db:"deposit_amount"`
	DepositToken    *string `structs:"deposit_token" db:"deposit_token"`
	Receiver        *string `structs:"receiver" db:"receiver"`
	WithdrawalToken *string `structs:"withdrawal_token" db:"withdrawal_token"`
	DepositBlock    *int64  `structs:"deposit_block" db:"deposit_block"`

	WithdrawalTxHash  *string `structs:"withdrawal_tx_hash" db:"withdrawal_tx_hash"`
	WithdrawalChainId *string `structs:"withdrawal_chain_id" db:"withdrawal_chain_id"`
	WithdrawalAmount  *string `structs:"withdrawal_amount" db:"withdrawal_amount"`

	IsWrappedToken *bool `structs:"is_wrapped_token" db:"is_wrapped_token"`

	SubmitStatus resources.SubmitWithdrawalStatus `structs:"submit_status" db:"submit_status"`
	Signature    *string                          `structs:"signature" db:"signature"`
}

func (d Deposit) Reprocessable() bool {
	return d.Status == resources.WithdrawalStatus_FAILED ||
		d.Status == resources.WithdrawalStatus_TX_FAILED
}

func (d Deposit) ToStatusResponse() *resources.CheckWithdrawalResponse {
	result := &resources.CheckWithdrawalResponse{
		Status: d.Status,
		DepositData: &resources.DepositData{
			EventIndex:       int64(d.TxEventId),
			Depositor:        d.Depositor,
			DepositAmount:    d.DepositAmount,
			WithdrawalAmount: d.WithdrawalAmount,
			DepositToken:     d.DepositToken,
			WithdrawalToken:  d.WithdrawalToken,
			Receiver:         d.Receiver,
			BlockNumber:      d.DepositBlock,
			Signature:        d.Signature,
			IsWrapped:        d.IsWrappedToken,
		},
		DepositTransaction: &resources.Transaction{
			Hash:    d.TxHash,
			ChainId: d.ChainId,
		},
		SubmitStatus: d.SubmitStatus,
	}

	if d.WithdrawalTxHash != nil && d.WithdrawalChainId != nil {
		result.WithdrawalTransaction = &resources.Transaction{
			Hash:    *d.WithdrawalTxHash,
			ChainId: *d.WithdrawalChainId,
		}
	}

	return result
}

func (d Deposit) ToTransaction() bridgetypes.Transaction {
	tx := bridgetypes.Transaction{
		DepositTxHash:    d.TxHash,
		DepositTxIndex:   uint64(d.TxEventId),
		DepositChainId:   d.ChainId,
		WithdrawalTxHash: stringOrEmpty(d.WithdrawalTxHash),
		Depositor:        stringOrEmpty(d.Depositor),
		// TODO: separate for deposit/withdrawal amount when added to bridge core
		Amount:            stringOrEmpty(d.WithdrawalAmount),
		DepositToken:      stringOrEmpty(d.DepositToken),
		Receiver:          stringOrEmpty(d.Receiver),
		WithdrawalToken:   stringOrEmpty(d.WithdrawalToken),
		WithdrawalChainId: stringOrEmpty(d.WithdrawalChainId),
		Signature:         stringOrEmpty(d.Signature),
	}

	if d.DepositBlock != nil {
		tx.DepositBlock = uint64(*d.DepositBlock)
	}

	return tx
}

type DepositData struct {
	DepositIdentifier
	DestinationChainId string

	SourceAddress      string
	DestinationAddress string

	DepositAmount    *big.Int
	WithdrawalAmount *big.Int

	TokenAddress            string
	DestinationTokenAddress string

	IsWrappedToken bool
	Signature      []byte

	Block int64
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
