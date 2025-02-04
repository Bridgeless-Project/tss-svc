package finalizer

import (
	"context"
	"fmt"
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

	//TODO: add session proposer creds to define whether party need to submit tx to core

	errChan chan error
	done    chan struct{}

	logger *logan.Entry
}

func NewEVMFinalizer(db database.DepositsQ, core *core.Connector, logger *logan.Entry) *EvmFinalizer {
	return &EvmFinalizer{
		db:      db,
		core:    core,
		errChan: make(chan error),
		done:    make(chan struct{}),
		logger:  logger,
	}
}

func (ef *EvmFinalizer) Run(ctx context.Context) error {
	go ef.saveAndBroadcast(ctx)

	// listen for ctx and errors
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err, ok := <-ef.errChan:
		if !ok {
			ef.logger.Debug("error chanel is closed")
			return nil
		}
		if err != nil {
			if updErr := ef.db.UpdateStatus(ef.withdrawalData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED); updErr != nil {
				return errors.Wrap(updErr, "finalization failed")
			}
			return errors.Wrap(err, "finalization failed")
		}
	case <-ef.done:
		return nil
	}

	return nil
}

func (ef *EvmFinalizer) saveAndBroadcast(ctx context.Context) {
	signature := hexutil.Encode(append(ef.signature.Signature, ef.signature.SignatureRecovery...))
	ef.logger.Info(fmt.Sprintf("got signature: %s", signature))

	err := ef.db.UpdateSignature(ef.withdrawalData.DepositIdentifier(), signature)
	if err != nil {
		ef.errChan <- err
		return
	}

	dep, err := ef.db.Get(ef.withdrawalData.DepositIdentifier())
	if err != nil {
		ef.errChan <- err
		return
	}

	// TODO: add checking if local party is a session proposer
	if err = ef.core.SubmitDeposits(ctx, dep.ToTransaction()); err != nil {
		ef.errChan <- err
		return
	}

	// send done signal
	ef.done <- struct{}{}
}

func (ef *EvmFinalizer) WithData(withdrawalData *withdrawal.EvmWithdrawalData) *EvmFinalizer {
	ef.withdrawalData = withdrawalData
	return ef
}

func (ef *EvmFinalizer) WithSignature(sig *common.SignatureData) *EvmFinalizer {
	ef.signature = sig
	return ef
}
