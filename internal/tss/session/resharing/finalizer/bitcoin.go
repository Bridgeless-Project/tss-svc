package finalizer

import (
	"bytes"
	"context"
	"crypto/ecdsa"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/wire"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/bitcoin"
	resharingCons "github.com/hyle-team/tss-svc/internal/tss/session/resharing/consensus"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type BitcoinResharingFinalizer struct {
	tssPub []byte
	client *bitcoin.Client

	data               *resharingCons.SigningData
	signatures         []*common.SignatureData
	localPartyProposer bool

	errChan chan error
	result  string

	logger *logan.Entry
}

func NewBitcoinResharingFinalizer(client *bitcoin.Client, pubKey *ecdsa.PublicKey, logger *logan.Entry) *BitcoinResharingFinalizer {
	return &BitcoinResharingFinalizer{
		client:  client,
		errChan: make(chan error),
		logger:  logger,
		tssPub:  ethcrypto.CompressPubkey(pubKey),
	}
}

func (f *BitcoinResharingFinalizer) WithData(data *resharingCons.SigningData) *BitcoinResharingFinalizer {
	f.data = data
	return f
}

func (f *BitcoinResharingFinalizer) WithSignatures(signatures []*common.SignatureData) *BitcoinResharingFinalizer {
	f.signatures = signatures
	return f
}

func (f *BitcoinResharingFinalizer) WithLocalPartyProposer(proposer bool) *BitcoinResharingFinalizer {
	f.localPartyProposer = proposer
	return f
}

func (f *BitcoinResharingFinalizer) Finalize(ctx context.Context) (string, error) {
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

func (f *BitcoinResharingFinalizer) finalize() {
	defer close(f.errChan)

	tx := wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(f.data.ProposalData.SerializedTx)); err != nil {
		f.errChan <- errors.Wrap(err, "failed to deserialize transaction")
		return
	}
	if err := bitcoin.InjectSignatures(&tx, f.signatures, f.tssPub); err != nil {
		f.errChan <- errors.Wrap(err, "failed to inject signatures")
		return
	}

	withdrawalTxHash := bridge.HexPrefix + tx.TxHash().String()
	f.result = withdrawalTxHash

	if !f.localPartyProposer {
		return
	}

	if _, err := f.client.SendSignedTransaction(&tx); err != nil {
		f.errChan <- errors.Wrap(err, "failed to broadcast finalized transaction")
		return
	}
}
