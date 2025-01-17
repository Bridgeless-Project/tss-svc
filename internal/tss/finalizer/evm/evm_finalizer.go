package evm

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	finalizerTypes "github.com/hyle-team/tss-svc/internal/tss/finalizer"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"sync"
)

type finalizer struct {
	chainClient evm.BridgeProxy
	// TODO: add Bridge core connector

	chainId string // to get finalizer for specific chain

	data    []byte
	rawData db.DepositData
	db      db.DepositsQ

	wg      *sync.WaitGroup
	errChan chan error
	err     error

	logger *logan.Entry
}

func newEVMFinalizer(chainClient evm.BridgeProxy, db db.DepositsQ, logger *logan.Entry) finalizerTypes.Finalizer {
	return &finalizer{
		wg:          &sync.WaitGroup{},
		chainClient: chainClient,
		db:          db,
		errChan:     make(chan error, 1),
		logger:      logger,
	}
}

func EVMFinalizer(chainClient types.Proxy, db db.DepositsQ, logger *logan.Entry) (finalizerTypes.Finalizer, error) {
	proxy, ok := chainClient.(evm.BridgeProxy)
	if !ok {
		return nil, errors.Wrap(errors.New("invalid proxy type"), "failed finalizer initialization")
	}
	return newEVMFinalizer(proxy, db, logger), nil
}
func (zf *finalizer) Run(ctx context.Context, data []byte, signatureData *common.SignatureData, deposit db.DepositData) error {
	if data == nil || signatureData == nil {
		return errors.Wrap(errors.New("invalid data"), "failed finalizer initialization")
	}
	zf.logger.Info("Starting finalization")
	// Configure ctx with timeout
	boundedCtx, cancel := context.WithTimeout(ctx, session.BoundaryFinalizeSession)
	defer cancel()

	// Configure finalizer for new data
	zf.data = data
	deposit.Signature = signatureData.Signature
	zf.rawData = deposit
	zf.logger.Info("configured data for finalization")

	zf.run(boundedCtx)
	return zf.waitFor()
}

func (zf *finalizer) run(ctx context.Context) {
	zf.wg.Add(2)
	// Save deposit signature
	go func() {
		defer zf.wg.Done()
		err := zf.db.SetDepositSignature(zf.rawData)
		if err != nil {
			zf.errChan <- err
			return
		}

		// Using core connector pass withdrawal tx info to Bridge core
		// TODO: Implement passing withdrawal data to Bridge core

	}()
	go zf.listen(ctx)
}
func (zf *finalizer) listen(ctx context.Context) {
	defer func() {
		close(zf.errChan)
		zf.wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			zf.err = errors.Wrap(ctx.Err(), "finalization timed out")
			return
		case err, ok := <-zf.errChan:
			if !ok {
				zf.logger.Debug("error chanel is closed")
				return
			}
			if err != nil {
				zf.err = errors.Wrap(err, "finalization failed")
				return
			}
			continue
		}
	}
}

func (zf *finalizer) waitFor() error {
	zf.wg.Wait()
	zf.logger.Info("finalizer finished")
	return zf.err
}
