package finalizer

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/db"
)

type FinalizersRepository interface {
	Finalizer(chainId string) (Finalizer, error)
	SupportsChain(chainId string) bool
}

type Finalizer interface {
	Run(ctx context.Context, data []byte, signatureData *common.SignatureData, deposit db.DepositData) error
}
