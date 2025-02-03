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
	"sync"
)

type EvmFinalizer struct {
	wg *sync.WaitGroup

	withdrawalData *withdrawal.EvmWithdrawalData
	signature      *common.SignatureData

	db   database.DepositsQ
	core *core.Connector

	//TODO: add session proposer creds to define whether party need to submit tx to core

	errChan chan error

	err    error
	logger *logan.Entry
}

func NewEVMFinalizer(db database.DepositsQ, core *core.Connector, logger *logan.Entry) *EvmFinalizer {
	return &EvmFinalizer{
		wg:      &sync.WaitGroup{},
		db:      db,
		core:    core,
		errChan: make(chan error, 1),
		logger:  logger,
	}
}

func (ef *EvmFinalizer) Run(ctx context.Context) error {
	ef.wg.Add(2)

	go ef.saveAndBroadcast()
	go ef.listen(ctx)

	ef.wg.Wait()

	return ef.err
}

func (ef *EvmFinalizer) listen(ctx context.Context) {
	defer func() {
		ef.wg.Done()
		ctx.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			ef.err = errors.Wrap(ctx.Err(), "finalization timed out")
			return
		case err, ok := <-ef.errChan:
			if !ok {
				ef.logger.Debug("error chanel is closed")
				return
			}
			if err != nil {
				ef.err = errors.Wrap(err, "finalization failed")
				ef.db.UpdateStatus(ef.withdrawalData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED)
				return
			}
			continue
		}
	}
}

func (ef *EvmFinalizer) saveAndBroadcast() {
	defer ef.wg.Done()
	signature := hexutil.Encode(append(ef.signature.Signature, ef.signature.SignatureRecovery...))
	ef.logger.Info(fmt.Sprintf("got signature: %s", signature))

	ef.errChan <- ef.db.UpdateSignature(ef.withdrawalData.DepositIdentifier(), signature)

	dep, err := ef.db.Get(ef.withdrawalData.DepositIdentifier())
	if err != nil {
		ef.errChan <- err
		return
	}

	// TODO: add checking if local party is a session proposer
	ef.errChan <- ef.core.SubmitDeposits(dep.ToTransaction())

}

func (ef *EvmFinalizer) WithData(withdrawalData *withdrawal.EvmWithdrawalData) *EvmFinalizer {
	ef.withdrawalData = withdrawalData
	return ef
}

func (ef *EvmFinalizer) WithSignature(sig *common.SignatureData) *EvmFinalizer {
	ef.signature = sig
	return ef
}
