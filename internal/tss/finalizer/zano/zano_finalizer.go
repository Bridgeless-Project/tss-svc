package zano

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge/client/zano"
	"github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"sync"
)

type finalizer struct {
	chainClient zano.BridgeProxy
	// TODO: add Bridge core connector
	// TODO: add Zano network connection

	chainId string // to get finalizer for specific chain

	data    []byte
	rawData db.DepositData
	db      db.DepositsQ

	wg      *sync.WaitGroup
	errChan chan error
	err     error

	logger *logan.Entry
}

func newZanoFinalizer(chainClient zano.BridgeProxy, db db.DepositsQ, logger *logan.Entry) *finalizer {
	return &finalizer{
		wg:          &sync.WaitGroup{},
		chainClient: chainClient,
		db:          db,
		errChan:     make(chan error, 4),
	}
}
func ZanoFinalizer(chainClient types.Proxy, db db.DepositsQ, logger *logan.Entry) (*finalizer, error) {
	proxy, ok := chainClient.(zano.BridgeProxy)
	if !ok {
		return nil, errors.Wrap(errors.New("invalid proxy type"), "failed finalizer initialization")
	}
	return newZanoFinalizer(proxy, db, logger), nil
}
func (zf *finalizer) Run(ctx context.Context, data []byte, signatureData *common.SignatureData, deposit db.DepositData) error {
	// Configure ctx with timeout
	boundedCtx, cancel := context.WithTimeout(ctx, session.BoundaryFinalizeSession)
	defer cancel()

	// Configure finalizer for new data
	zf.data = data
	deposit.Signature = signatureData.Signature
	zf.rawData = deposit

	zf.run(boundedCtx)
	return zf.waitFor()
}

func (zf *finalizer) run(ctx context.Context) {
	zf.wg.Add(2)
	// Save deposit signature
	go func() {
		defer zf.wg.Done()
		zf.errChan <- zf.db.SetDepositSignature(zf.rawData)

		// Using core connector pass withdrawal tx info to Bridge core
		// TODO: Implement passing withdrawal data to chain

		// Pass withdrawal tx to network
		// TODO: using network connector pass tx to network

	}()
	go zf.listen(ctx)
}
func (zf *finalizer) listen(ctx context.Context) {
	defer func() {
		close(zf.errChan)
		zf.wg.Done()
	}()
	select {
	case <-ctx.Done():
		zf.err = errors.Wrap(ctx.Err(), "finalization timed out")
	case err, ok := <-zf.errChan:
		if !ok {
			return
		}
		if err != nil {
			zf.err = errors.Wrap(err, "finalization failed")
		}
	}
}

func (zf *finalizer) waitFor() error {
	zf.wg.Wait()
	return zf.err
}
