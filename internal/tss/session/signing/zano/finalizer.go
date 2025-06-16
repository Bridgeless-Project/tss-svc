package zano

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/bnb-chain/tss-lib/v2/common"
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
		f.logger.Info("finalization finished")

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (f *Finalizer) finalize(ctx context.Context) {
	withdrawalTxHash := bridge.HexPrefix + f.withdrawalData.ProposalData.TxId
	if err := f.db.UpdateWithdrawalTx(f.withdrawalData.DepositIdentifier(), withdrawalTxHash); err != nil {
		f.errChan <- errors.Wrap(err, "failed to update withdrawal tx")
		return
	}

	dep, err := f.db.Get(f.withdrawalData.DepositIdentifier())
	if err != nil {
		f.errChan <- errors.Wrap(err, "failed to get deposit")
		return
	}

	signedTx := zano.SignedTransaction{
		Signature: zano.EncodeSignature(f.signature),
		UnsignedTransaction: zano.UnsignedTransaction{
			ExpectedTxHash: f.withdrawalData.ProposalData.TxId,
			FinalizedTx:    f.withdrawalData.ProposalData.FinalizedTx,
			Data:           f.withdrawalData.ProposalData.UnsignedTx,
		},
	}
	encodedTx := signedTx.Encode()

	if err = f.core.SubmitDeposits(ctx, dep.ToTransaction(&encodedTx)); err != nil {
		f.errChan <- errors.Wrap(err, "failed to submit deposit")
		return
	}

	if !f.sessionLeader {
		f.errChan <- nil
		return
	}

	_, err = f.client.SendSignedTransaction(signedTx)
	if err != nil {
		f.errChan <- errors.Wrap(err, "failed to emit signed transaction")
		return
	}

	f.errChan <- nil
}
