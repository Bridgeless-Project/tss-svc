package bitcoin

import (
	"bytes"
	"context"
	"crypto/ecdsa"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/wire"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/bitcoin"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Finalizer struct {
	tssPub []byte
	client *bitcoin.Client

	data          *SigningData
	signatures    []*common.SignatureData
	sessionLeader bool

	errChan chan error
	result  string

	logger *logan.Entry
}

func NewFinalizer(
	client *bitcoin.Client,
	pubKey *ecdsa.PublicKey,
	logger *logan.Entry,
	sessionLeader bool,
) *Finalizer {
	return &Finalizer{
		client:        client,
		errChan:       make(chan error),
		logger:        logger,
		tssPub:        ethcrypto.CompressPubkey(pubKey),
		sessionLeader: sessionLeader,
	}
}

func (f *Finalizer) WithData(data *SigningData) *Finalizer {
	f.data = data
	return f
}

func (f *Finalizer) WithSignatures(signatures []*common.SignatureData) *Finalizer {
	f.signatures = signatures
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

	// ignoring error here, as the mempool tx can be already observed by the wallet
	_ = f.client.LockOutputs(tx)

	if !f.sessionLeader {
		return
	}

	if _, err := f.client.SendSignedTransaction(&tx); err != nil {
		f.errChan <- errors.Wrap(err, "failed to broadcast finalized transaction")
		return
	}
}
