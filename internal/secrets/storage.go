package secrets

import (
	"crypto/tls"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

type TssShares struct {
	Share      *keygen.LocalPartySaveData
	FrostShare *frostkeygen.Config
}

type Storage interface {
	GetKeygenPreParams() (*keygen.LocalPreParams, error)
	SaveKeygenPreParams(params *keygen.LocalPreParams) error

	GetCoreAccount() (*core.Account, error)
	SaveCoreAccount(account *core.Account) error

	SaveTssShare(data interface{}) error
	GetTssShare() (interface{}, int, error)
	GetTssShares() (*TssShares, error)

	SaveLocalPartyTlsCertificate(rawCert, rawKey []byte) error
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}
