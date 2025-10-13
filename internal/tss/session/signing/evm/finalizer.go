package evm

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	withdrawalData *withdrawal.EvmWithdrawalData
	signature      *common.SignatureData

	db   database.DepositsQ
	core *coreConnector.Connector

	sessionLeader bool

	errChan chan error

	logger *logan.Entry
}

func NewFinalizer(
	db database.DepositsQ,
	core *coreConnector.Connector,
	logger *logan.Entry,
	sessionLeader bool) *Finalizer {
	return &Finalizer{
		db:            db,
		core:          core,
		errChan:       make(chan error),
		logger:        logger,
		sessionLeader: sessionLeader,
	}
}

func (ef *Finalizer) WithData(withdrawalData *withdrawal.EvmWithdrawalData) *Finalizer {
	ef.withdrawalData = withdrawalData
	return ef
}

func (ef *Finalizer) WithSignature(sig *common.SignatureData) *Finalizer {
	ef.signature = sig
	return ef
}

func (ef *Finalizer) Finalize(ctx context.Context) error {
	ef.logger.Info("finalization started")
	go ef.finalize(ctx)

	// listen for ctx and errors
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err := <-ef.errChan:
		ef.logger.Info("finalization finished")

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (ef *Finalizer) finalize(ctx context.Context) {
	signature := convertToEthSignature(ef.signature)
	if err := ef.db.UpdateProcessed(database.ProcessedDepositData{
		Identifier: ef.withdrawalData.DepositIdentifier(),
		Signature:  &signature,
	}); err != nil {
		ef.errChan <- errors.Wrap(err, "failed to update signature")
		return
	}

	//if err := ef.db.UpdateSignature(ef.withdrawalData.DepositIdentifier(), signature); err != nil {
	//	ef.errChan <- errors.Wrap(err, "failed to update signature")
	//	return
	//}

	//dep, err := ef.db.Get(ef.withdrawalData.DepositIdentifier())
	//if err != nil {
	//	ef.errChan <- errors.Wrap(err, "failed to get deposit")
	//	return
	//}
	//
	//if err = ef.core.SubmitDeposits(ctx, dep.ToTransaction(nil)); err != nil {
	//	ef.errChan <- errors.Wrap(err, "failed to submit deposit")
	//	return
	//}

	ef.errChan <- nil
}

func convertToEthSignature(sig *common.SignatureData) string {
	rawSig := append(sig.Signature, sig.SignatureRecovery...)
	rawSig[64] += 27

	return hexutil.Encode(rawSig)
}
