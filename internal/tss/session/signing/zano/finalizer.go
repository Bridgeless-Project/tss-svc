package zano

import (
	"context"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/zano"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	coreConnector "github.com/hyle-team/tss-svc/internal/core/connector"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	withdrawalData *withdrawal.ZanoWithdrawalData
	signature      *common.SignatureData

	db   database.DepositsQ
	core *coreConnector.Connector

	client *zano.Client

	sessionLeader bool

	errChan chan error
	logger  *logan.Entry
}

func NewFinalizer(
	db database.DepositsQ,
	core *coreConnector.Connector,
	client *zano.Client,
	logger *logan.Entry,
	sessionLeader bool) *Finalizer {
	return &Finalizer{
		db:            db,
		core:          core,
		errChan:       make(chan error),
		logger:        logger,
		client:        client,
		sessionLeader: sessionLeader,
	}
}

func (f *Finalizer) WithData(withdrawalData *withdrawal.ZanoWithdrawalData) *Finalizer {
	f.withdrawalData = withdrawalData
	return f
}

func (f *Finalizer) WithSignature(signature *common.SignatureData) *Finalizer {
	f.signature = signature
	return f
}

func (f *Finalizer) Finalize(ctx context.Context) error {
	f.logger.Info("finalization started")
	go f.finalize(ctx)

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err := <-f.errChan:
		if err == nil {
			f.logger.Info("finalization finished")
			return nil
		}

		if updErr := f.db.UpdateStatus(f.withdrawalData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED); updErr != nil {
			return errors.Wrap(err, "failed to finalize withdrawal and update its status")
		}

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (f *Finalizer) finalize(ctx context.Context) {
	if err := f.db.UpdateWithdrawalTx(f.withdrawalData.DepositIdentifier(), f.withdrawalData.ProposalData.TxId); err != nil {
		f.errChan <- errors.Wrap(err, "failed to update withdrawal tx")
		return
	}

	if !f.sessionLeader {
		f.errChan <- nil
		return
	}

	_, err := f.client.EmitAssetSigned(zano.SignedTransaction{
		Signature: zano.EncodeSignature(f.signature),
		UnsignedTransaction: zano.UnsignedTransaction{
			ExpectedTxHash: f.withdrawalData.ProposalData.TxId,
			FinalizedTx:    f.withdrawalData.ProposalData.FinalizedTx,
			Data:           f.withdrawalData.ProposalData.UnsignedTx,
		},
	})
	if err != nil {
		f.errChan <- errors.Wrap(err, "failed to emit signed transaction")
		return
	}

	dep, err := f.db.Get(f.withdrawalData.DepositIdentifier())
	if err != nil {
		f.errChan <- errors.Wrap(err, "failed to get deposit")
		return
	}

	if err = f.core.SubmitDeposits(ctx, dep.ToTransaction()); err != nil {
		f.errChan <- errors.Wrap(err, "failed to submit deposit")
		return
	}

	f.errChan <- nil
}
