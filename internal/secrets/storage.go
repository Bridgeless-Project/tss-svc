package secrets

import (
	"crypto/tls"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"

	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

type TssShares struct {
	Share      *keygen.LocalPartySaveData
	FrostShare *frostkeygen.Config
}

type TssShareKey string

const (
	TssShareKeyECDSA     TssShareKey = "tss_shares/ecdsa"
	TssShareKeyFROST     TssShareKey = "tss_shares/frost"
	TssShareKeyTemporary TssShareKey = "tss_share_temp"
)

type Storage interface {
	GetKeygenPreParams() (*keygen.LocalPreParams, error)
	SaveKeygenPreParams(params *keygen.LocalPreParams) error

	GetCoreAccount() (*core.Account, error)
	SaveCoreAccount(account *core.Account) error

	SaveTssShare(key TssShareKey, data interface{}) error
	GetTssShare() (interface{}, int, error)
	GetTssShares() (*TssShares, error)

	// TODO: implement the FROST key gen
	GetTemporaryTssShare() (*keygen.LocalPartySaveData, error)

	SaveLocalPartyTlsCertificate(rawCert, rawKey []byte) error
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}
