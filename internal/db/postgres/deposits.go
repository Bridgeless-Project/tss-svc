package pg

import (
	"database/sql"
	"strings"

	"github.com/hyle-team/tss-svc/internal/types"

	"github.com/Masterminds/squirrel"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/kit/pgdb"
)

const (
	depositsTable   = "deposits"
	depositsTxHash  = "tx_hash"
	depositsTxNonce = "tx_nonce"
	depositsChainId = "chain_id"
	depositsId      = "id"

	depositsDepositor        = "depositor"
	depositsDepositAmount    = "deposit_amount"
	depositsWithdrawalAmount = "withdrawal_amount"
	depositsDepositToken     = "deposit_token"
	depositsReceiver         = "receiver"
	depositsWithdrawalToken  = "withdrawal_token"
	depositsDepositBlock     = "deposit_block"

	depositsWithdrawalChainId = "withdrawal_chain_id"

	depositWithdrawalStatus = "withdrawal_status"

	depositIsWrappedToken = "is_wrapped_token"

	depositSignature = "signature"
)

type depositsQ struct {
	db       *pgdb.DB
	selector squirrel.SelectBuilder
}

func (d *depositsQ) New() db.DepositsQ {
	return NewDepositsQ(d.db.Clone())
}

func (d *depositsQ) Insert(deposit db.Deposit) (int64, error) {
	stmt := squirrel.
		Insert(depositsTable).
		SetMap(map[string]interface{}{
			depositsTxHash:           deposit.TxHash,
			depositsTxNonce:          deposit.TxNonce,
			depositsChainId:          deposit.ChainId,
			depositWithdrawalStatus:  deposit.WithdrawalStatus,
			depositsDepositAmount:    *deposit.DepositAmount,
			depositsWithdrawalAmount: *deposit.WithdrawalAmount,
			depositsReceiver:         strings.ToLower(*deposit.Receiver),
			depositsDepositBlock:     *deposit.DepositBlock,
			depositIsWrappedToken:    *deposit.IsWrappedToken,
			// can be 0x00... in case of native ones
			depositsDepositToken: strings.ToLower(*deposit.DepositToken),
			depositsDepositor:    strings.ToLower(*deposit.Depositor),
			// can be 0x00... in case of native ones
			depositsWithdrawalToken:   strings.ToLower(*deposit.WithdrawalToken),
			depositsWithdrawalChainId: *deposit.WithdrawalChainId,
		}).
		Suffix("RETURNING id")

	var id int64
	if err := d.db.Get(&id, stmt); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			err = db.ErrAlreadySubmitted
		}

		return id, err
	}

	return id, nil
}

func (d *depositsQ) Get(identifier db.DepositIdentifier) (*db.Deposit, error) {
	var deposit db.Deposit
	err := d.db.Get(&deposit, d.selector.Where(squirrel.Eq{
		depositsTxHash:  identifier.TxHash,
		depositsTxNonce: identifier.TxNonce,
		depositsChainId: identifier.ChainId,
	}))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &deposit, err
}

func (d *depositsQ) Select(selector db.DepositsSelector) ([]db.Deposit, error) {
	query := d.applySelector(selector, d.selector)
	var deposits []db.Deposit
	if err := d.db.Select(&deposits, query); err != nil {
		return nil, err
	}

	return deposits, nil
}

func NewDepositsQ(db *pgdb.DB) db.DepositsQ {
	return &depositsQ{
		db:       db.Clone(),
		selector: squirrel.Select("*").From(depositsTable),
	}
}

func (d *depositsQ) Transaction(f func() error) error {
	return d.db.Transaction(f)
}

func (d *depositsQ) applySelector(selector db.DepositsSelector, sql squirrel.SelectBuilder) squirrel.SelectBuilder {
	if len(selector.Ids) > 0 {
		sql = sql.Where(squirrel.Eq{depositsId: selector.Ids})
	}

	if selector.Submitted != nil {
		sql = sql.Where(squirrel.Eq{depositWithdrawalStatus: types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING})
	}

	return sql
}
