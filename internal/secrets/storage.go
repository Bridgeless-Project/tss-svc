package secrets

import (
	"crypto/tls"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"

	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

type TssShares struct {
	Share      *keygen.LocalPartySaveData
	FrostShare *frostkeygen.Config
}

type TssShareKey string

type Storage interface {

	// TODO: make frost friendly
	GetKeygenPreParams() (*keygen.LocalPreParams, error)
	SaveKeygenPreParams(params *keygen.LocalPreParams) error

	GetCoreAccount() (*core.Account, error)
	SaveCoreAccount(account *core.Account) error

	SaveTssShare(key TssShareKey, data []byte) error
	LoadTssShare(share tss.Share) error

	GetTemporaryTssShare(share tss.Share) error

	SaveLocalPartyTlsCertificate(rawCert, rawKey []byte) error
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}
