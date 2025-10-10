package utxo

import (
	"bytes"
	"context"
	"crypto/ecdsa"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	withdrawalData *withdrawal.UtxoWithdrawalData
	signatures     []*common.SignatureData

	tssPub *ecdsa.PublicKey

	db   database.DepositsQ
	core *coreConnector.Connector

	client client.Client

	sessionLeader bool

	errChan chan error
	logger  *logan.Entry
}

func NewFinalizer(
	db database.DepositsQ,
	core *coreConnector.Connector,
	client client.Client,
	pubKey *ecdsa.PublicKey,
	logger *logan.Entry,
	sessionLeader bool,
) *Finalizer {
	return &Finalizer{
		db:            db,
		core:          core,
		errChan:       make(chan error),
		logger:        logger,
		client:        client,
		tssPub:        pubKey,
		sessionLeader: sessionLeader,
	}
}

func (f *Finalizer) WithData(withdrawalData *withdrawal.UtxoWithdrawalData) *Finalizer {
	f.withdrawalData = withdrawalData
	return f
}

func (f *Finalizer) WithSignatures(signatures []*common.SignatureData) *Finalizer {
	f.signatures = signatures
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
	tx := &wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(f.withdrawalData.ProposalData.SerializedTx)); err != nil {
		f.errChan <- errors.Wrap(err, "failed to deserialize transaction")
		return
	}
	if err := f.client.UtxoHelper().InjectSignatures(tx, f.signatures, f.tssPub); err != nil {
		f.errChan <- errors.Wrap(err, "failed to inject signatures")
		return
	}

	withdrawalTxHash := bridge.HexPrefix + f.client.UtxoHelper().TxHash(tx)
	//if err := f.db.UpdateWithdrawalTx(f.withdrawalData.DepositIdentifier(), withdrawalTxHash); err != nil {
	//	f.errChan <- errors.Wrap(err, "failed to update withdrawal tx")
	//	return
	//}

	// ignoring error here, as the mempool tx can be already observed by the wallet
	_ = f.client.LockOutputs(tx)

	//dep, err := f.db.Get(f.withdrawalData.DepositIdentifier())
	//if err != nil {
	//	f.errChan <- errors.Wrap(err, "failed to get deposit")
	//	return
	//}

	encodedTx := utils.EncodeTransaction(tx)

	if err := f.db.UpdateProcessed(database.ProcessedDepositData{
		Identifier: f.withdrawalData.DepositIdentifier(),
		TxData:     &encodedTx,
		TxHash:     &withdrawalTxHash,
	}); err != nil {
		f.errChan <- errors.Wrap(err, "failed to update signature")
		return
	}

	//if err = f.core.SubmitDeposits(ctx, dep.ToTransaction(&encodedTx)); err != nil {
	//	f.errChan <- errors.Wrap(err, "failed to submit deposit")
	//	return
	//}

	if !f.sessionLeader {
		f.errChan <- nil
		return
	}

	if _, err := f.client.SendSignedTransaction(tx); err != nil {
		f.errChan <- errors.Wrap(err, "failed to send signed transaction")
		return
	}

	f.errChan <- nil
}
