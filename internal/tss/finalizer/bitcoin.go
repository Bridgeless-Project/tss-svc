package finalizer

import (
	"bytes"
	"context"
	"crypto/ecdsa"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/wire"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/bitcoin"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	core "github.com/hyle-team/tss-svc/internal/core/connector"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type BitcoinFinalizer struct {
	withdrawalData *withdrawal.BitcoinWithdrawalData
	signatures     []*common.SignatureData

	tssPub []byte

	db   database.DepositsQ
	core *core.Connector

	client *bitcoin.Client

	localPartyProposer bool

	errChan chan error
	logger  *logan.Entry
}

func NewBitcoinFinalizer(
	db database.DepositsQ,
	core *core.Connector,
	client *bitcoin.Client,
	pubKey *ecdsa.PublicKey,
	logger *logan.Entry,
) *BitcoinFinalizer {
	return &BitcoinFinalizer{
		db:      db,
		core:    core,
		errChan: make(chan error),
		logger:  logger,
		client:  client,
		tssPub:  ethcrypto.CompressPubkey(pubKey),
	}
}

func (f *BitcoinFinalizer) WithData(withdrawalData *withdrawal.BitcoinWithdrawalData) *BitcoinFinalizer {
	f.withdrawalData = withdrawalData
	return f
}

func (f *BitcoinFinalizer) WithSignatures(signatures []*common.SignatureData) *BitcoinFinalizer {
	f.signatures = signatures
	return f
}

func (f *BitcoinFinalizer) WithLocalPartyProposer(proposer bool) *BitcoinFinalizer {
	f.localPartyProposer = proposer
	return f
}

func (f *BitcoinFinalizer) Finalize(ctx context.Context) error {
	f.logger.Info("finalization started")
	go f.finalize()

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

func (f *BitcoinFinalizer) finalize() {
	tx := wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(f.withdrawalData.ProposalData.SerializedTx)); err != nil {
		f.errChan <- errors.Wrap(err, "failed to deserialize transaction")
		return
	}
	if err := bitcoin.InjectSignatures(&tx, f.signatures, f.tssPub); err != nil {
		f.errChan <- errors.Wrap(err, "failed to inject signatures")
		return
	}

	// TODO: save to db

	if !f.localPartyProposer {
		f.errChan <- nil
		return
	}

	_, err := f.client.SendSignedTransaction(&tx)
	if err != nil {
		f.errChan <- errors.Wrap(err, "failed to send signed transaction")
		return
	}

	// TODO: save to core

	f.errChan <- nil

}
