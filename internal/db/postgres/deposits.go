package pg

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
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
	depositsReferralId       = "referral_id"

	depositsWithdrawalChainId = "withdrawal_chain_id"
	depositsWithdrawalTxHash  = "withdrawal_tx_hash"

	depositsWithdrawalStatus = "withdrawal_status"

	depositsIsWrappedToken   = "is_wrapped_token"
	depositsCommissionAmount = "commission_amount"

	depositsSignature   = "signature"
	depositsTxData      = "tx_data"
	depositsSubmitted   = "submitted"
	depositsDistributed = "distributed"
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
			depositsReceiver:         deposit.Receiver,
			depositsDepositBlock:     deposit.DepositBlock,
			depositsIsWrappedToken:   deposit.IsWrappedToken,
			// can be 0x00... in case of native ones
			depositsDepositToken: deposit.DepositToken,
			depositsDepositor:    deposit.Depositor,
			// can be 0x00... in case of native ones
			depositsWithdrawalToken:   deposit.WithdrawalToken,
			depositsWithdrawalChainId: deposit.WithdrawalChainId,
			depositsCommissionAmount:  deposit.CommissionAmount,
			depositsReferralId:        deposit.ReferralId,

			depositsSubmitted:   false,
			depositsDistributed: deposit.Distributed,
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
		Set(depositsSubmitted, true).
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

func (d *depositsQ) UpdateProcessed(data db.ProcessedDepositData) error {
	query := squirrel.Update(depositsTable)

	if data.TxHash != nil {
		query = query.Set(depositsWithdrawalTxHash, *data.TxHash)
	}
	if data.Signature != nil {
		query = query.Set(depositsSignature, *data.Signature)
	}
	if data.TxData != nil {
		query = query.Set(depositsTxData, *data.TxData)
	}

	query = query.
		Set(depositsWithdrawalStatus, types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED).
		Where(identifierToPredicate(data.Identifier))

	return d.db.Exec(query)
}

func (d *depositsQ) UpdateSubmittedStatus(identifier db.DepositIdentifier, submitted bool) error {
	query := squirrel.Update(depositsTable).
		Set(depositsSubmitted, submitted).
		Where(identifierToPredicate(identifier))

	return d.db.Exec(query)
}

func (d *depositsQ) UpdateDistributedStatus(identifier db.DepositIdentifier, distributed bool) error {
	query := squirrel.Update(depositsTable).
		Set(depositsDistributed, distributed).
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
	if selector.WithdrawalChainId != nil {
		sql = sql.Where(squirrel.Eq{depositsWithdrawalChainId: *selector.WithdrawalChainId})
	}
	if selector.Status != nil {
		sql = sql.Where(squirrel.Eq{depositsWithdrawalStatus: *selector.Status})
	}
	if selector.NotSubmitted {
		sql = sql.Where(squirrel.Eq{depositsSubmitted: false})
	}
	if selector.Distributed {
		sql = sql.Where(squirrel.Eq{depositsDistributed: true})
	}
	if selector.NotDistributed {
		sql = sql.Where(squirrel.Eq{depositsDistributed: false})
	}
	if selector.One {
		sql = sql.OrderBy(fmt.Sprintf("%s ASC", depositsId)).Limit(1)
	}
	if selector.SortAscending {
		sql = sql.OrderBy(fmt.Sprintf("%s ASC", depositsId))
	}
	if selector.Limit > 0 {
		sql = sql.Limit(selector.Limit)
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
			depositsCommissionAmount: deposit.CommissionAmount,
			depositsReceiver:         strings.ToLower(deposit.Receiver),
			depositsDepositBlock:     deposit.DepositBlock,
			depositsIsWrappedToken:   deposit.IsWrappedToken,
			// can be 0x00... in case of native ones
			depositsDepositToken: strings.ToLower(deposit.DepositToken),
			depositsDepositor:    deposit.Depositor,
			// can be 0x00... in case of native ones
			depositsWithdrawalToken:   strings.ToLower(deposit.WithdrawalToken),
			depositsWithdrawalChainId: deposit.WithdrawalChainId,
			depositsWithdrawalTxHash:  deposit.WithdrawalTxHash,
			depositsSignature:         deposit.Signature,
			depositsWithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
			depositsReferralId:        deposit.ReferralId,
			depositsTxData:            deposit.TxData,
			depositsSubmitted:         true,
			depositsDistributed:       true,
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

func (d *depositsQ) UpdateSignedBatch(signed []db.SignedDeposit) error {
	var (
		signatures = make(pq.StringArray, len(signed))
		ids        = make(pq.Int64Array, len(signed))
	)
	for i, deposit := range signed {
		ids[i], signatures[i] = deposit.Id, deposit.Signature
	}

	const query = `
		UPDATE deposits
		SET
			status = $1,
			signature = unnested_data.signature
		FROM (
			SELECT unnest($2::bigint[]) AS id, unnest($3::text[]) AS signature
		) AS unnested_data
		WHERE deposits.id = unnested_data.id;
`
	return d.db.ExecRaw(
		query,
		types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
		ids,
		signatures,
	)
}
