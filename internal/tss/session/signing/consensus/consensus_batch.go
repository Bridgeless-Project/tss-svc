package consensus

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
)

var _ consensus.Mechanism[withdrawal.DepositSigningData] = &BatchDepositConsensusMechanism[withdrawal.DepositSigningData]{}

type BatchDepositConsensusMechanism[T withdrawal.DepositSigningData] struct {
	depositSelector db.DepositsSelector
	depositsQ       db.DepositsQ
	constructor     withdrawal.Constructor[T]
	fetcher         deposit.Fetcher
}

func NewBatchDepositConsensusMechanism[T withdrawal.DepositSigningData](
	chainId string,
	depositsQ db.DepositsQ,
	constructor withdrawal.Constructor[T],
	fetcher *deposit.Fetcher,
) *BatchDepositConsensusMechanism[T] {
	return &BatchDepositConsensusMechanism[T]{
		depositSelector: db.DepositsSelector{
			WithdrawalChainId: &chainId,
			Status:            new(types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING),
			Distributed:       true, // only consider deposits that have been distributed to other parties
			Limit:             100,
			One:               false,
		},
		depositsQ:   depositsQ,
		constructor: constructor,
		fetcher:     *fetcher,
	}
}

type ErrMissingDeposits struct {
	MissingIDs []db.DepositIdentifier
}

func (e *ErrMissingDeposits) Error() string {
	return "missing deposits in proposal"
}

func (c *BatchDepositConsensusMechanism[T]) FormProposalData() (*T, error) {
	unsignedDeposits, err := c.depositsQ.Select(c.depositSelector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposits")
	}
	if len(unsignedDeposits) == 0 {
		return nil, nil
	}

	proposalData, err := c.constructor.FormSigningData(unsignedDeposits...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to form proposal data")
	}

	return proposalData, nil
}

func (c *BatchDepositConsensusMechanism[T]) VerifyProposedData(data T) error {
	unsignedDeposits := data.DepositIdentifiers()

	if len(unsignedDeposits) == 0 {
		return nil
	}

	selector := db.DepositsSelector{
		Identifiers: unsignedDeposits,
	}

	existingDeposits, err := c.depositsQ.Select(selector)
	if err != nil {
		return errors.Wrap(err, "failed to get deposits")
	}

	foundMap := make(map[db.DepositIdentifier]db.Deposit, len(existingDeposits))
	for _, dep := range existingDeposits {
		key := db.DepositIdentifier{
			TxHash:  dep.TxHash,
			TxNonce: dep.TxNonce,
			ChainId: dep.ChainId,
		}
		foundMap[key] = dep
	}

	depositsToValidate := make([]db.Deposit, 0, len(unsignedDeposits))
	missingIDs := make([]db.DepositIdentifier, 0)

	for _, id := range unsignedDeposits {
		unsignedDeposit, exists := foundMap[id]
		if !exists {
			missingIDs = append(missingIDs, id)
			continue
		}
		if unsignedDeposit.WithdrawalStatus != types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING {
			return errors.New("deposit is not in pending status")
		}
		depositsToValidate = append(depositsToValidate, unsignedDeposit)
	}

	if len(missingIDs) > 0 {
		return &ErrMissingDeposits{
			MissingIDs: missingIDs,
		}
	}

	isValid, err := c.constructor.IsValid(data, depositsToValidate...)
	if err != nil {
		return errors.Wrap(err, "failed to validate proposal data")
	}
	if !isValid {
		return errors.New("proposal data is invalid")
	}

	return nil
}
