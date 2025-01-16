package zano

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge/client/zano"
	"github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

type Finalizer struct {
	chainClient types.Proxy
	// TODO: add Bridge core connector
	// TODO: add Zano network connection

	data    []byte
	rawData db.DepositData
	db      db.DepositsQ
}

func NewFinalizer(chainClient types.Proxy, data []byte, signatureData *common.SignatureData, db db.DepositsQ, deposit db.DepositData) *Finalizer {
	deposit.Signature = signatureData.Signature
	return &Finalizer{
		chainClient: chainClient,
		data:        data,
		db:          db,
		rawData:     deposit,
	}
}

func (zf *Finalizer) Run(_ context.Context) error {
	zfcProxy, ok := zf.chainClient.(zano.BridgeProxy)
	if !ok {
		return errors.Wrap(errors.New("invalid proxy type"), "finalization failed")
	}
	zf.chainClient = zfcProxy

	// Save the data with signature to db
	err := zf.db.SetDepositSignature(zf.rawData)
	if err != nil {
		return errors.Wrap(err, "finalization failed")
	}

	// Using core connector pass withdrawal tx info to Bridge core
	// TODO: Implement passing data to core

	// TODO: Implement passing withdrawal data to chain
	return nil

}
