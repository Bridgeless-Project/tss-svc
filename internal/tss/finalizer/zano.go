package finalizer

import (
	"context"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/zano"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	core "github.com/hyle-team/tss-svc/internal/core/connector"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type ZanoFinalizer struct {
	withdrawalData *withdrawal.ZanoWithdrawalData
	signature      *common.SignatureData

	db   database.DepositsQ
	core *core.Connector

	client *zano.Client

	localPartyProposer bool

	errChan chan error
	logger  *logan.Entry
}

func NewZanoFinalizer(db database.DepositsQ, core *core.Connector, client *zano.Client, logger *logan.Entry) *ZanoFinalizer {
	return &ZanoFinalizer{
		db:      db,
		core:    core,
		errChan: make(chan error),
		logger:  logger,
		client:  client,
	}
}

func (f *ZanoFinalizer) WithData(withdrawalData *withdrawal.ZanoWithdrawalData) *ZanoFinalizer {
	f.withdrawalData = withdrawalData
	return f
}

func (f *ZanoFinalizer) WithSignature(signature *common.SignatureData) *ZanoFinalizer {
	f.signature = signature
	return f
}

func (f *ZanoFinalizer) WithLocalPartyProposer(proposer bool) *ZanoFinalizer {
	f.localPartyProposer = proposer
	return f
}

func (f *ZanoFinalizer) Finalize(ctx context.Context) error {
	go f.finalize()

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "finalization timed out")
	case err := <-f.errChan:
		if err == nil {
			return nil
		}

		if updErr := f.db.UpdateStatus(f.withdrawalData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED); updErr != nil {
			return errors.Wrap(err, "failed to finalize withdrawal and update its status")
		}

		return errors.Wrap(err, "failed to finalize withdrawal")
	}
}

func (f *ZanoFinalizer) finalize() {
	if err := f.db.UpdateWithdrawalTx(f.withdrawalData.DepositIdentifier(), f.withdrawalData.ProposalData.TxId); err != nil {
		f.errChan <- errors.Wrap(err, "failed to update withdrawal tx")
		return
	}

	if !f.localPartyProposer {
		f.errChan <- nil
		return
	}

	rawSig := append(f.signature.Signature, f.signature.SignatureRecovery...)
	zanoSig := encodeToZanoSignature(rawSig)
	_, err := f.client.EmitAssetSigned(zano.SignedTransaction{
		Signature: zanoSig,
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

	f.errChan <- nil
}

func encodeToZanoSignature(signature []byte) string {
	if len(signature) == 0 {
		return ""
	}

	encoded := hexutil.Encode(signature)
	// stripping redundant hex-prefix and recovery byte (two hex-characters)
	strippedSignature := encoded[2 : len(encoded)-2]

	return strippedSignature
}
