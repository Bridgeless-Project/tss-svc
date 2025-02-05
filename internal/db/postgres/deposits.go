package pg

import (
	"database/sql"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
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
	depositsWithdrawalTxHash  = "withdrawal_tx_hash"

	depositsWithdrawalStatus = "withdrawal_status"

	depositsIsWrappedToken = "is_wrapped_token"

	depositsSignature = "signature"
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
			depositsWithdrawalStatus: deposit.WithdrawalStatus,
			depositsDepositAmount:    deposit.DepositAmount,
			depositsWithdrawalAmount: deposit.WithdrawalAmount,
			depositsReceiver:         strings.ToLower(*deposit.Receiver),
			depositsDepositBlock:     deposit.DepositBlock,
			depositsIsWrappedToken:   deposit.IsWrappedToken,
			// can be 0x00... in case of native ones
			depositsDepositToken: strings.ToLower(*deposit.DepositToken),
			depositsDepositor:    deposit.Depositor,
			// can be 0x00... in case of native ones
			depositsWithdrawalToken:   strings.ToLower(*deposit.WithdrawalToken),
			depositsWithdrawalChainId: deposit.WithdrawalChainId,
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

func (d *depositsQ) Exists(check db.DepositExistenceCheck) (bool, error) {
	var deposit db.Deposit
	err := d.db.Get(&deposit, d.selector.Where(existenceToPredicate(check)))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

func (d *depositsQ) Get(identifier db.DepositIdentifier) (*db.Deposit, error) {
	var deposit db.Deposit
	err := d.db.Get(&deposit, d.selector.Where(identifierToPredicate(identifier)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &deposit, err
}

func identifierToPredicate(identifier db.DepositIdentifier) squirrel.Eq {
	return squirrel.Eq{
		depositsTxHash:  identifier.TxHash,
		depositsTxNonce: identifier.TxNonce,
		depositsChainId: identifier.ChainId,
	}
}

func existenceToPredicate(check db.DepositExistenceCheck) squirrel.Eq {
	predicate := squirrel.Eq{}
	if check.ByTxHash != nil {
		predicate[depositsTxHash] = *check.ByTxHash
	}

	if check.ByTxNonce != nil {
		predicate[depositsTxNonce] = *check.ByTxNonce
	}

	if check.ByChainId != nil {
		predicate[depositsChainId] = *check.ByChainId
	}

	return predicate
}

func (d *depositsQ) GetWithSelector(selector db.DepositsSelector) (*db.Deposit, error) {
	query := d.applySelector(selector, d.selector)
	var deposit db.Deposit
	err := d.db.Get(&deposit, query)
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

func (d *depositsQ) UpdateWithdrawalDetails(identifier db.DepositIdentifier, hash *string, signature *string) error {
	query := squirrel.Update(depositsTable).
		Set(depositsWithdrawalTxHash, hash).
		Set(depositsSignature, signature).
		Set(depositsWithdrawalStatus, types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED).
		Where(identifierToPredicate(identifier))

	return d.db.Exec(query)
}

func (d *depositsQ) UpdateSignature(identifier db.DepositIdentifier, sig string) error {
	query := squirrel.Update(depositsTable).
		Set(depositsWithdrawalStatus, types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED).
		Where(identifierToPredicate(identifier))

	return d.db.Exec(query)
}

func (d *depositsQ) UpdateStatus(identifier db.DepositIdentifier, status types.WithdrawalStatus) error {
	query := squirrel.Update(depositsTable).
		Set(depositsWithdrawalStatus, status).
		Where(identifierToPredicate(identifier))

	return d.db.Exec(query)
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

	if selector.ChainId != nil {
		sql = sql.Where(squirrel.Eq{depositsChainId: *selector.ChainId})
	}

	if selector.Status != nil {
		sql = sql.Where(squirrel.Eq{depositsWithdrawalStatus: *selector.Status})
	}

	if selector.One {
		sql = sql.Limit(1)
	}

	return sql
}

func (d *depositsQ) InsertProcessedDeposit(deposit db.Deposit) (int64, error) {
	stmt := squirrel.
		Insert(depositsTable).
		SetMap(map[string]interface{}{
			depositsTxHash:           deposit.TxHash,
			depositsTxNonce:          deposit.TxNonce,
			depositsChainId:          deposit.ChainId,
			depositsDepositAmount:    deposit.DepositAmount,
			depositsWithdrawalAmount: deposit.WithdrawalAmount,
			depositsReceiver:         strings.ToLower(*deposit.Receiver),
			depositsDepositBlock:     deposit.DepositBlock,
			depositsIsWrappedToken:   deposit.IsWrappedToken,
			// can be 0x00... in case of native ones
			depositsDepositToken: strings.ToLower(*deposit.DepositToken),
			depositsDepositor:    deposit.Depositor,
			// can be 0x00... in case of native ones
			depositsWithdrawalToken:   strings.ToLower(*deposit.WithdrawalToken),
			depositsWithdrawalChainId: deposit.WithdrawalChainId,
			depositsWithdrawalTxHash:  deposit.WithdrawalTxHash,
			depositsSignature:         *deposit.Signature,
			depositsWithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
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

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
