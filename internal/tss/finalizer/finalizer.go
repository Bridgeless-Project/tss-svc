package finalizer

import (
	"github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/tss/finalizer/zano"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type finalizerRepo struct {
	finalizers map[string]Finalizer
}

func NewFinalizersRepo(proxiesRepo map[string]types.Proxy, db db.DepositsQ, logger *logan.Entry) (FinalizersRepository, error) {
	finalizers := make(map[string]Finalizer)
	for chainId, proxy := range proxiesRepo {
		var finalizer Finalizer
		var err error
		switch proxy.Type() {
		case types.ChainTypeZano:
			finalizer, err = zano.ZanoFinalizer(proxy, db, logger)
			if err != nil {
				return nil, errors.Wrap(err, "zano finalizer init failed")
			}
		// TODO: add EVM case
		// TODO: add BTC case
		default:
			return nil, errors.Errorf("unknown chain type: %s", proxy.Type())
		}
		finalizers[chainId] = finalizer

	}
	return &finalizerRepo{finalizers: finalizers}, nil
}

func (f *finalizerRepo) Finalizer(chainId string) (Finalizer, error) {
	if !f.SupportsChain(chainId) {
		return nil, errors.Wrap(errors.New("chain is not supported"), "failed to find finalizer")
	}
	return f.finalizers[chainId], nil
}

func (f *finalizerRepo) SupportsChain(chainId string) bool {
	_, ok := f.finalizers[chainId]
	return ok
}
