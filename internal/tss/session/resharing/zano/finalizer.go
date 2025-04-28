package zano

import (
	"context"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/zano"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	data      *SigningData
	signature *common.SignatureData

	client *zano.Client

	sessionLeader bool
	errChan       chan error
	result        string

	logger *logan.Entry
}

func NewFinalizer(client *zano.Client, logger *logan.Entry, sessionLeader bool) *Finalizer {
	return &Finalizer{
		client:        client,
		errChan:       make(chan error),
		logger:        logger,
		sessionLeader: sessionLeader,
	}
}

func (f *Finalizer) WithData(data *SigningData) *Finalizer {
	f.data = data
	return f
}

func (f *Finalizer) WithSignature(signature *common.SignatureData) *Finalizer {
	f.signature = signature
	return f
}

func (f *Finalizer) Finalize(ctx context.Context) (string, error) {
	f.logger.Info("finalization started")
	go f.finalize()

	select {
	case <-ctx.Done():
		f.logger.Info("finalization cancelled")
		return "", ctx.Err()
	case err := <-f.errChan:
		f.logger.Info("finalization finished")
		return f.result, err
	}
}

func (f *Finalizer) finalize() {
	defer close(f.errChan)

	f.result = bridge.HexPrefix + f.data.ProposalData.TxId

	if !f.sessionLeader {
		return
	}

	_, err := f.client.SendSignedTransaction(zano.SignedTransaction{
		Signature: zano.EncodeSignature(f.signature),
		UnsignedTransaction: zano.UnsignedTransaction{
			ExpectedTxHash: f.data.ProposalData.TxId,
			FinalizedTx:    f.data.ProposalData.FinalizedTx,
			Data:           f.data.ProposalData.UnsignedTx,
		},
	})

	f.errChan <- err
}
