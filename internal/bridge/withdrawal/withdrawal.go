package withdrawal

import (
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/tss/consensus"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

type DepositSigningData interface {
	consensus.SigningData
	DepositIdentifier() db.DepositIdentifier
}

type SigDataFormer[T DepositSigningData] interface {
	FormSigningData(deposit db.Deposit) (*T, error)
}

type SigDataPayloader[T DepositSigningData] interface {
	FromPayload(payload *anypb.Any) (*T, error)
}

type SigDataValidator[T DepositSigningData] interface {
	IsValid(data T, deposit db.Deposit) (bool, error)
}

type Constructor[T DepositSigningData] interface {
	SigDataFormer[T]
	SigDataValidator[T]
	SigDataPayloader[T]
}

var _ consensus.Mechanism[DepositSigningData] = &ConsensusMechanism[DepositSigningData]{}

type ConsensusMechanism[T DepositSigningData] struct {
	depositSelector db.DepositsSelector
	depositsQ       db.DepositsQ
	constructor     Constructor[T]
	fetcher         bridge.DepositFetcher
}

func NewConsensusMechanism[T DepositSigningData](
	chainId string,
	depositsQ db.DepositsQ,
	constructor Constructor[T],
	fetcher *bridge.DepositFetcher,
) *ConsensusMechanism[T] {
	var pendingWithdrawalStatus = types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING
	return &ConsensusMechanism[T]{
		depositSelector: db.DepositsSelector{
			WithdrawalChainId: &chainId,
			Status:            &pendingWithdrawalStatus,
			One:               true,
		},
		depositsQ:   depositsQ,
		constructor: constructor,
		fetcher:     *fetcher,
	}
}

func (c *ConsensusMechanism[T]) FormProposalData() (*T, error) {
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

func (c *ConsensusMechanism[T]) FromPayload(payload *anypb.Any) (*T, error) {
	return c.constructor.FromPayload(payload)
}

func (c *ConsensusMechanism[T]) VerifyProposedData(data T) error {
	unsignedDeposit, err := c.depositsQ.Get(data.DepositIdentifier())
	if err != nil {
		return errors.Wrap(err, "failed to get deposit")
	}
	if unsignedDeposit == nil {
		unsignedDeposit, err = c.fetcher.FetchDeposit(data.DepositIdentifier())
		if err != nil {
			return errors.Wrap(err, "failed to fetch deposit")
		}
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
