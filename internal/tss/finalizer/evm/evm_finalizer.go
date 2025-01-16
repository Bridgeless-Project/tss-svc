package evm

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

type EVMFinalizer struct {
	chainClient types.Proxy
	// TODO: add Bridge core connector

	data    []byte
	rawData db.DepositData
	db      db.DepositsQ
}

func NewEVMFinalizer(chainClient types.Proxy, data []byte, signatureData *common.SignatureData, db db.DepositsQ, deposit db.DepositData) *EVMFinalizer {
	deposit.Signature = signatureData.Signature
	return &EVMFinalizer{
		chainClient: chainClient,
		data:        data,
		db:          db,
		rawData:     deposit,
	}
}

func (ef *EVMFinalizer) Run(_ context.Context) error {
	evmProxy, ok := ef.chainClient.(evm.BridgeProxy)
	if !ok {
		return errors.Wrap(errors.New("invalid proxy type"), "finalization failed")
	}
	ef.chainClient = evmProxy

	// Save the data with signature to db
	err := ef.db.SetDepositSignature(ef.rawData)
	if err != nil {
		return errors.Wrap(err, "finalization failed")
	}

	// Using core connector pass withdrawal tx info to Bridge core
	// TODO: Implement passing data to core
	return nil

}
