package pg

import (
	"database/sql"
	"encoding/hex"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/lib/pq"
	"strings"

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

func (d *depositsQ) SetWithdrawalTxs(txs ...db.WithdrawalTx) error {
	if len(txs) == 0 {
		return nil
	}

	var (
		hashes = make(pq.StringArray, len(txs))
		chains = make(pq.StringArray, len(txs))
		ids    = make(pq.Int64Array, len(txs))
	)
	for i, tx := range txs {
		hashes[i] = strings.ToLower(tx.TxHash)
		chains[i] = tx.ChainId
		ids[i] = tx.DepositId
	}

	const query string = `
UPDATE deposits
SET
    status = $1,
    withdrawal_tx_hash = unnested_db.tx_hash,
    withdrawal_chain_id = unnested_db.chain_id
FROM (
	SELECT unnest($2::text[]) as tx_hash,
    	   unnest($3::text[]) as chain_id,
    	   unnest($4::bigint[]) as deposit_id
) as unnested_db
WHERE deposits.id = unnested_db.deposit_id
`

	return d.db.ExecRaw(query, types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING, hashes, chains, ids)
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

func (d *depositsQ) UpdateWithdrawalStatus(status types.WithdrawalStatus, ids ...int64) error {
	stmt := squirrel.Update(depositsTable).
		Set(depositWithdrawalStatus, status).
		Where(squirrel.Eq{depositsId: ids})

	return d.db.Exec(stmt)
}

func (d *depositsQ) SetDepositSignature(data db.DepositData) error {
	fields := map[string]interface{}{
		depositSignature:        strings.ToLower(hex.EncodeToString([]byte(data.Signature))),
		depositWithdrawalStatus: types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
	}

	return d.db.Exec(squirrel.Update(depositsTable).Where(
		squirrel.Eq{
			depositsTxHash:  data.TxHash,
			depositsTxNonce: data.TxNonce,
			depositsChainId: data.ChainId,
		},
	).SetMap(fields))
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
