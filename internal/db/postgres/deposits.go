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

func (d *depositsQ) UpdateStatus(status types.WithdrawalStatus, identifier ...db.DepositIdentifier) error {
	if len(identifier) == 0 {
		return nil
	}

	var (
		hashes   = make(pq.StringArray, len(identifier))
		nonces   = make(pq.Int64Array, len(identifier))
		chainIds = make(pq.StringArray, len(identifier))
	)

	for i, id := range identifier {
		hashes[i] = id.TxHash
		nonces[i] = id.TxNonce
		chainIds[i] = id.ChainId
	}

	const query = `
        UPDATE deposits
        SET withdrawal_status = $1
        FROM (
            SELECT 
                unnest($2::text[]) AS hash, 
                unnest($3::bigint[]) AS nonce, 
                unnest($4::text[]) AS chain_id
        ) AS unnested_data
        WHERE deposits.tx_hash = unnested_data.hash
          AND deposits.tx_nonce = unnested_data.nonce
          AND deposits.chain_id = unnested_data.chain_id;
    `

	return d.db.ExecRaw(query,
		types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
		hashes,
		nonces,
		chainIds,
	)
}

func (d *depositsQ) UpdateProcessed(data ...db.ProcessedDepositData) error {
	if len(data) == 0 {
		return nil
	}

	var (
		hashes   = make(pq.StringArray, len(data))
		nonces   = make(pq.Int64Array, len(data))
		chainIds = make(pq.StringArray, len(data))

		withdrawalHashes = make([]*string, len(data))
		txData           = make([]*string, len(data))
		signatures       = make([]*string, len(data))
		proofs           = make([]*string, len(data))
	)

	for i, deposit := range data {
		hashes[i] = deposit.Identifier.TxHash
		nonces[i] = deposit.Identifier.TxNonce
		chainIds[i] = deposit.Identifier.ChainId

		withdrawalHashes[i] = deposit.TxHash
		txData[i] = deposit.TxData
		signatures[i] = deposit.Signature
		proofs[i] = deposit.MerkleProof
	}

	const query = `
        UPDATE deposits
        SET
            withdrawal_status  = $1,
            withdrawal_tx_hash = unnested_data.withdrawal_tx_hash,
            tx_data            = unnested_data.tx_data,
            signature          = unnested_data.signature,
            merkle_proof       = unnested_data.merkle_proof
        FROM (
            SELECT
                unnest($2::text[])   AS tx_hash,
                unnest($3::bigint[]) AS tx_nonce,
                unnest($4::text[])   AS chain_id,
                unnest($5::text[])   AS withdrawal_tx_hash,
                unnest($6::text[])   AS tx_data,
                unnest($7::text[])   AS signature,
                unnest($8::text[])   AS merkle_proof
        ) AS unnested_data
        WHERE deposits.tx_hash  = unnested_data.tx_hash
          AND deposits.tx_nonce = unnested_data.tx_nonce
          AND deposits.chain_id = unnested_data.chain_id;`

	return d.db.ExecRaw(query,
		types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED,
		hashes,
		nonces,
		chainIds,
		pq.Array(withdrawalHashes),
		pq.Array(txData),
		pq.Array(signatures),
		pq.Array(proofs),
	)
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
	if len(selector.Identifiers) > 0 {
		hashes := make(pq.StringArray, len(selector.Identifiers))
		nonces := make(pq.Int64Array, len(selector.Identifiers))
		chainIds := make(pq.StringArray, len(selector.Identifiers))

		for i, id := range selector.Identifiers {
			hashes[i] = id.TxHash
			nonces[i] = id.TxNonce
			chainIds[i] = id.ChainId
		}

		sql = sql.Where(
			`(tx_hash, tx_nonce, chain_id) IN (
        	SELECT unnest(?::text[]), unnest(?::bigint[]), unnest(?::text[]))`,
			hashes, nonces, chainIds,
		)
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
			withdrawal_status = $1,
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
