package vault

import (
	"context"
	"crypto/tls"
	"encoding/json"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	client "github.com/hashicorp/vault/api"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/pkg/errors"
)

const (
	valueKey     = "value"
	keyPreParams = "keygen_preparams"
	keyAccount   = "core_account"
	keyTssShare  = "tss_share"
	keyTlsCert   = "tls_cert"
	tlsCertData  = "cert_data"
	tlsKeyData   = "key_data"
)

type Storage struct {
	client *client.KVv2
}

func NewStorage(client *client.KVv2) secrets.Storage {
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
		return nil, errors.New("data not found")
	}

	return kvData.Data, nil
}

func (s *Storage) store(path string, value map[string]interface{}) error {
	if _, err := s.client.Put(context.Background(), path, value); err != nil {
		return errors.Wrap(err, "failed to save data")
	}

	return nil
}

func (s *Storage) GetKeygenPreParams() (*keygen.LocalPreParams, error) {
	data, err := s.load(keyPreParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load preparams")
	}

	val, ok := data[valueKey].(string)
	if !ok {
		return nil, errors.New("preparams value not found")
	}

	params := new(keygen.LocalPreParams)
	if err = json.Unmarshal([]byte(val), params); err != nil {
		return nil, errors.Wrap(err, "failed to decode preparams")
	}

	return params, nil
}

func (s *Storage) SaveKeygenPreParams(params *keygen.LocalPreParams) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "failed to marshal preparams")
	}

	return s.store(keyPreParams, map[string]interface{}{
		valueKey: string(raw),
	})
}

func (s *Storage) SaveTssShare(data *keygen.LocalPartySaveData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal share data")
	}

	return s.store(keyTssShare, map[string]interface{}{
		valueKey: string(raw),
	})
}

func (s *Storage) GetCoreAccount() (*core.Account, error) {
	kvData, err := s.load(keyAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load account")
	}

	val, ok := kvData[valueKey].(string)
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
		valueKey: hexutil.Encode(account.PrivateKey().Bytes()),
	})
}

func (s *Storage) GetTssShare() (*keygen.LocalPartySaveData, error) {
	kvData, err := s.load(keyTssShare)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load share data")
	}
	val, ok := kvData[valueKey].(string)
	if !ok {
		return nil, errors.New("share data not found")
	}
	data := new(keygen.LocalPartySaveData)
	err = json.Unmarshal([]byte(val), data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode share data")
	}
	return data, nil
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
