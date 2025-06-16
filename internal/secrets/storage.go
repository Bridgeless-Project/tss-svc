package secrets

import (
	"crypto/tls"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
)

type Storage interface {
	GetKeygenPreParams() (*keygen.LocalPreParams, error)
	SaveKeygenPreParams(params *keygen.LocalPreParams) error

	GetCoreAccount() (*core.Account, error)
	SaveCoreAccount(account *core.Account) error

	SaveTssShare(data *keygen.LocalPartySaveData) error
	GetTssShare() (*keygen.LocalPartySaveData, error)

	SaveLocalPartyTlsCertificate(rawCert, rawKey []byte) error
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}
