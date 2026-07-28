package vault

import (
	"context"
	"crypto/tls"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	client "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

const (
	keyAccount     = "core_account"
	keyTssShareSet = "tss_shares/current"

	valueVaultKey = "value"

	keyTlsCert  = "tls_cert"
	tlsCertData = "cert_data"
	tlsKeyData  = "key_data"
)

var errDataNotFound = errors.New("data not found")

type Storage struct {
	client KVStore
}

type KVStore interface {
	Get(ctx context.Context, secretPath string) (*client.KVSecret, error)
	Put(ctx context.Context, secretPath string, data map[string]interface{}, opts ...client.KVOption) (*client.KVSecret, error)
}

func NewStorage(client *client.KVv2) secrets.Storage {
	return NewStorageFromKV(client)
}

func NewStorageFromKV(client KVStore) *Storage {
	return &Storage{
		client: client,
	}
}

func (s *Storage) load(path string) (map[string]interface{}, error) {
	kvData, err := s.client.Get(context.Background(), path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load data")
	}
	if kvData == nil {
		return nil, errDataNotFound
	}

	return kvData.Data, nil
}

func (s *Storage) store(path string, value map[string]interface{}) error {
	if _, err := s.client.Put(context.Background(), path, value); err != nil {
		return errors.Wrap(err, "failed to save data")
	}

	return nil
}

func (s *Storage) GetKeygenPreParams(params tss.PreParams) error {
	data, err := s.load(params.GetVaultPath())
	if err != nil {
		return errors.Wrap(err, "failed to load preparams")
	}

	if err = params.SetVaultData(data); err != nil {
		return errors.Wrap(err, "failed to set preparams data")
	}

	return nil
}

func (s *Storage) SaveKeygenPreParams(params tss.PreParams) error {
	raw, err := params.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to marshal preparams")
	}

	return s.SaveTssShare(secrets.TssShareKey(params.GetVaultPath()), raw)
}

func (s *Storage) SaveTssShare(key secrets.TssShareKey, bytes []byte) error {
	if strings.HasPrefix(string(key), "tss_shares/") {
		return s.SaveTssShares(map[secrets.TssShareKey][]byte{key: bytes})
	}
	return s.store(string(key), map[string]interface{}{
		valueVaultKey: bytes,
	})
}

func (s *Storage) SaveTssShares(shares map[secrets.TssShareKey][]byte) error {
	if len(shares) == 0 {
		return errors.New("no TSS shares to save")
	}
	values := make(map[string]interface{}, len(shares))
	if current, err := s.load(keyTssShareSet); err == nil {
		for key, value := range current {
			values[key] = value
		}
	} else if !isSecretNotFound(err) {
		return errors.Wrap(err, "failed to load current TSS share set")
	}
	for key, value := range shares {
		values[string(key)] = value
	}
	return errors.Wrap(s.store(keyTssShareSet, values), "failed to atomically save TSS shares")
}

func (s *Storage) GetCoreAccount() (*core.Account, error) {
	kvData, err := s.load(keyAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load account")
	}

	val, ok := kvData[valueVaultKey].(string)
	if !ok {
		return nil, errors.New("account value not found")
	}

	account, err := core.NewAccount(val)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse account")
	}

	return account, nil
}

func (s *Storage) SaveCoreAccount(account *core.Account) error {
	return s.store(keyAccount, map[string]interface{}{
		valueVaultKey: hexutil.Encode(account.PrivateKey().Bytes()),
	})
}

func (s *Storage) LoadTssShare(share tss.Share) error {
	if shares, batchErr := s.load(keyTssShareSet); batchErr == nil {
		if value, ok := shares[share.GetVaultPath()]; ok {
			if err := share.SetVaultData(map[string]interface{}{valueVaultKey: value}); err != nil {
				return errors.Wrap(err, "failed to set batched share data")
			}
			return nil
		}
		// An active set is authoritative. Falling back to a legacy per-protocol
		// record here could resurrect a stale share after an atomic rotation.
		return errors.Wrapf(errDataNotFound, "active TSS share set has no %s", share.GetVaultPath())
	} else if !isSecretNotFound(batchErr) {
		return errors.Wrap(batchErr, "failed to load active TSS share set")
	}

	data, err := s.load(share.GetVaultPath())
	if err != nil {
		return errors.Wrap(err, "failed to load share data")
	}

	if err = share.SetVaultData(data); err != nil {
		return errors.Wrap(err, "failed to set share data")
	}

	return nil
}

func isSecretNotFound(err error) bool {
	if errors.Is(err, errDataNotFound) {
		return true
	}
	var responseErr *client.ResponseError
	return errors.As(err, &responseErr) && responseErr.StatusCode == 404
}

func (s *Storage) GetTemporaryTssShare(share tss.Share) error {
	kvData, err := s.load(string(secrets.TemporaryTssShareKey(share)))
	if err != nil {
		return errors.Wrap(err, "failed to load temporary share data")
	}

	return share.SetVaultData(kvData)
}

func (s *Storage) GetLocalPartyTlsCertificate() (*tls.Certificate, error) {
	kvData, err := s.load(keyTlsCert)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load tls certificate data")
	}
	rawCert, ok := kvData[tlsCertData].(string)
	if !ok {
		return nil, errors.New("tls certificate data not found")
	}
	rawKey, ok := kvData[tlsKeyData].(string)
	if !ok {
		return nil, errors.New("tls key data not found")
	}

	cert, err := tls.X509KeyPair([]byte(rawCert), []byte(rawKey))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tls certificate")
	}

	return &cert, nil
}

func (s *Storage) SaveLocalPartyTlsCertificate(rawCert, rawKey []byte) error {
	return s.store(keyTlsCert, map[string]interface{}{
		tlsCertData: string(rawCert),
		tlsKeyData:  string(rawKey),
	})

}
