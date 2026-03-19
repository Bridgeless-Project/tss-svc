package consensus

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
)

var _ consensus.Mechanism[withdrawal.DepositSigningData] = &SingleDepositConsensusMechanism[withdrawal.DepositSigningData]{}

type SingleDepositConsensusMechanism[T withdrawal.DepositSigningData] struct {
	depositSelector db.DepositsSelector
	depositsQ       db.DepositsQ
	constructor     withdrawal.Constructor[T]
	fetcher         deposit.Fetcher
}

func NewSingleDepositConsensusMechanism[T withdrawal.DepositSigningData](
	chainId string,
	depositsQ db.DepositsQ,
	constructor withdrawal.Constructor[T],
	fetcher *deposit.Fetcher,
) *SingleDepositConsensusMechanism[T] {
	var pendingWithdrawalStatus = types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING
	return &SingleDepositConsensusMechanism[T]{
		depositSelector: db.DepositsSelector{
			WithdrawalChainId: &chainId,
			Status:            &pendingWithdrawalStatus,
			Distributed:       true, // only consider deposits that have been distributed to other parties
			One:               true,
		},
		depositsQ:   depositsQ,
		constructor: constructor,
		fetcher:     *fetcher,
	}
}

func (c *SingleDepositConsensusMechanism[T]) FormProposalData() (*T, error) {
	unsignedDeposit, err := c.depositsQ.GetWithSelector(c.depositSelector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposit")
	}
	if unsignedDeposit == nil {
		return nil, nil
	}

	proposalData, err := c.constructor.FormSigningData(*unsignedDeposit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to form proposal data")
	}

	return proposalData, nil
}

func (c *SingleDepositConsensusMechanism[T]) VerifyProposedData(data T) error {
	if len(data.DepositIdentifiers()) == 0 {
		return errors.New("no deposit identifiers in proposal")
	}
	unsignedDeposit, err := c.depositsQ.Get(data.DepositIdentifiers()[0])
	if err != nil {
		return errors.Wrap(err, "failed to get deposit")
	}
	if unsignedDeposit == nil {
		unsignedDeposit, err = c.fetcher.FetchDeposit(data.DepositIdentifiers()[0])
		if err != nil {
			return errors.Wrap(err, "failed to fetch deposit")
		}
		unsignedDeposit.Distributed = true
		if _, err := c.depositsQ.Insert(*unsignedDeposit); err != nil {
			if !errors.Is(err, db.ErrAlreadySubmitted) {
				return errors.Wrap(err, "failed to save fetched deposit")
			}
		}
	}
	if unsignedDeposit.WithdrawalStatus != types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING {
		return errors.New("deposit is not in pending status")
	}

	isValid, err := c.constructor.IsValid(data, *unsignedDeposit)
	if err != nil {
		return errors.Wrap(err, "failed to validate proposal data")
	}
	if !isValid {
		return errors.New("proposal data is invalid")
	}

	return nil
}
