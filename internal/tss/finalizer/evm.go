package finalizer

import (
	"context"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	core "github.com/hyle-team/tss-svc/internal/core/connector"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type EvmFinalizer struct {
	withdrawalData *withdrawal.EvmWithdrawalData
	signature      *common.SignatureData

	db   database.DepositsQ
	core *core.Connector

	localPartyProposer bool

	errChan chan error

	logger *logan.Entry
}

func NewEVMFinalizer(db database.DepositsQ, core *core.Connector, logger *logan.Entry) *EvmFinalizer {
	return &EvmFinalizer{
		db:      db,
		core:    core,
		errChan: make(chan error),
		logger:  logger,
	}
}

func (ef *EvmFinalizer) WithData(withdrawalData *withdrawal.EvmWithdrawalData) *EvmFinalizer {
	ef.withdrawalData = withdrawalData
	return ef
}

func (ef *EvmFinalizer) WithSignature(sig *common.SignatureData) *EvmFinalizer {
	ef.signature = sig
	return ef
}

func (ef *EvmFinalizer) WithLocalPartyProposer(proposer bool) *EvmFinalizer {
	ef.localPartyProposer = proposer
	return ef
}

func (ef *EvmFinalizer) Finalize(ctx context.Context) error {
	go ef.finalize(ctx)

	// listen for ctx and errors
	select {
	case <-ctx.Done():
		// FIXME: should we update the status of the withdrawal?
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err := <-ef.errChan:
		if err == nil {
			return nil
		}

		if updErr := ef.db.UpdateStatus(ef.withdrawalData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED); updErr != nil {
			return errors.Wrap(err, "failed to finalize withdrawal and update its status")
		}

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (ef *EvmFinalizer) finalize(ctx context.Context) {
	signature := hexutil.Encode(append(ef.signature.Signature, ef.signature.SignatureRecovery...))
	if err := ef.db.UpdateSignature(ef.withdrawalData.DepositIdentifier(), signature); err != nil {
		ef.errChan <- err
		return
	}

	if !ef.localPartyProposer {
		ef.errChan <- nil
		return
	}

	dep, err := ef.db.Get(ef.withdrawalData.DepositIdentifier())
	if err != nil {
		ef.errChan <- err
		return
	}

	if err = ef.core.SubmitDeposits(ctx, dep.ToTransaction()); err != nil {
		ef.errChan <- errors.Wrap(err, "failed to submit deposit")
		return
	}

	ef.errChan <- nil
}
