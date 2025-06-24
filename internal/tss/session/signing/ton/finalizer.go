package ton

import (
	"context"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	coreConnector "github.com/hyle-team/tss-svc/internal/core/connector"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	withdrawalData *withdrawal.TonWithdrawalData
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

func (tf *Finalizer) WithData(withdrawalData *withdrawal.TonWithdrawalData) *Finalizer {
	tf.withdrawalData = withdrawalData
	return tf
}

func (tf *Finalizer) WithSignature(sig *common.SignatureData) *Finalizer {
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

func (tf *Finalizer) finalize(ctx context.Context) {
	signature := convertToEthSignature(tf.signature)
	if err := tf.db.UpdateSignature(tf.withdrawalData.DepositIdentifier(), signature); err != nil {
		tf.errChan <- errors.Wrap(err, "failed to update signature")
		return
	}

	dep, err := tf.db.Get(tf.withdrawalData.DepositIdentifier())
	if err != nil {
		tf.errChan <- errors.Wrap(err, "failed to get deposit")
		return
	}

	if err = tf.core.SubmitDeposits(ctx, dep.ToTransaction(nil)); err != nil {
		tf.errChan <- errors.Wrap(err, "failed to submit deposit")
		return
	}

	tf.errChan <- nil
}

func convertToEthSignature(sig *common.SignatureData) string {
	rawSig := append(sig.Signature, sig.SignatureRecovery...)
	rawSig[64] += 27

	return hexutil.Encode(rawSig)
}
