package solana

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
	withdrawalData *withdrawal.SolanaWithdrawalData
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

func (f *Finalizer) WithData(withdrawalData *withdrawal.SolanaWithdrawalData) *Finalizer {
	f.withdrawalData = withdrawalData
	return f
}

func (f *Finalizer) WithSignature(sig *common.SignatureData) *Finalizer {
	f.signature = sig
	return f
}

func (f *Finalizer) Finalize(ctx context.Context) error {
	f.logger.Info("finalization started")
	go f.finalize(ctx)

	// listen for ctx and errors
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err := <-f.errChan:
		f.logger.Info("finalization finished")

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (f *Finalizer) finalize(_ context.Context) {
	signature := convertToSolanaSignature(f.signature)
	if err := f.db.UpdateProcessed(database.ProcessedDepositData{
		Identifier: f.withdrawalData.DepositIdentifier(),
		Signature:  &signature,
	}); err != nil {
		f.errChan <- errors.Wrap(err, "failed to update signature")
		return
	}

	f.errChan <- nil
}

func convertToSolanaSignature(sig *common.SignatureData) string {
	rawSig := append(sig.Signature, sig.SignatureRecovery...)
	return hexutil.Encode(rawSig)
}
