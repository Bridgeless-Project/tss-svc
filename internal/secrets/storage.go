package secrets

import (
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/hyle-team/tss-svc/internal/core"
)

type Storage interface {
	GetKeygenPreParams() (*keygen.LocalPreParams, error)
	SaveKeygenPreParams(params *keygen.LocalPreParams) error

	GetCoreAccount() (*core.Account, error)
	SaveCoreAccount(account *core.Account) error

	SaveTssShare(data *keygen.LocalPartySaveData) error
}
