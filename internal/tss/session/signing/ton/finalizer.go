package ton

import (
	"context"

	tonchain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/ton"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	withdrawalData *withdrawal.TonWithdrawalData
	signature      tss.SignatureData

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

func (tf *Finalizer) WithData(withdrawalData *withdrawal.TonWithdrawalData) *Finalizer {
	tf.withdrawalData = withdrawalData
	return tf
}

func (tf *Finalizer) WithSignature(sig tss.SignatureData) *Finalizer {
	tf.signature = sig
	return tf
}

func (tf *Finalizer) Finalize(ctx context.Context) error {
	tf.logger.Info("finalization started")
	go tf.finalize(ctx)

	// listen for ctx and errors
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err := <-tf.errChan:
		tf.logger.Info("finalization finished")

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (tf *Finalizer) finalize(_ context.Context) {
	signature, err := tonchain.ConvertToTonSignature(tf.signature)
	if err != nil {
		tf.errChan <- errors.Wrap(err, "failed to convert signature")
		return
	}
	if err := tf.db.UpdateProcessed(database.ProcessedDepositData{
		Identifier: tf.withdrawalData.DepositIdentifiers()[0],
		Signature:  &signature,
	}); err != nil {
		tf.errChan <- errors.Wrap(err, "failed to update signature")
		return
	}

	tf.errChan <- nil
}
