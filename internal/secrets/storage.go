package secrets

import (
	"crypto/tls"
	"path"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
)

type TssShareKey string

const (
	TssShareKeyTemporary TssShareKey = "temp"
)

func TemporaryTssShareKey(share tss.Share) TssShareKey {
	return TssShareKey(path.Join(string(TssShareKeyTemporary), share.GetVaultPath()))
}

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

// BatchTssShareStorage atomically activates a set of protocol shares when the
// backing secret store supports a single-record update.
type BatchTssShareStorage interface {
	SaveTssShares(shares map[TssShareKey][]byte) error
}
