package secrets

import (
	"crypto/tls"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
)

type TssShareKey string

const (
	TssShareKeyTemporary TssShareKey = "temp/tss_shares/"
)

type Storage interface {
	GetKeygenPreParams(params tss.PreParams) error
	SaveKeygenPreParams(params tss.PreParams) error

	GetCoreAccount() (*core.Account, error)
	SaveCoreAccount(account *core.Account) error

	SaveTssShare(key TssShareKey, data []byte) error
	LoadTssShare(share tss.Share) error

	GetTemporaryTssShare(share tss.Share) error

	SaveLocalPartyTlsCertificate(rawCert, rawKey []byte) error
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}
