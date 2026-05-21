package vault

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	frostTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/fxamacker/cbor/v2"
	client "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
)

const (
	valueKey          = "value"
	protocolKey       = "protocol"
	encodingKey       = "encoding"
	keyPreParams      = "keygen_preparams"
	keyAccount        = "core_account"
	legacyKeyTssShare = "tss_share"
	keyTssShareECDSA  = string(secrets.TssShareKeyECDSA)
	keyTssShareFROST  = string(secrets.TssShareKeyFROST)

	keyTlsCert  = "tls_cert"
	tlsCertData = "cert_data"
	tlsKeyData  = "key_data"

	protocolFrost = "frost"
	encodingCBOR  = "cbor-base64"

	keyTssShareTemp = string(secrets.TssShareKeyTemporary)
)

type Storage struct {
	client kvStore
}

type kvStore interface {
	Get(ctx context.Context, secretPath string) (*client.KVSecret, error)
	Put(ctx context.Context, secretPath string, data map[string]interface{}, opts ...client.KVOption) (*client.KVSecret, error)
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

func (s *Storage) loadOptional(path string) (map[string]interface{}, bool, error) {
	kvData, err := s.client.Get(context.Background(), path)
	if err != nil {
		if errors.Is(err, client.ErrSecretNotFound) {
			return nil, false, nil
		}

		return nil, false, errors.Wrap(err, "failed to load data")
	}
	if kvData == nil {
		return nil, false, nil
	}

	return kvData.Data, true, nil
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

func (s *Storage) SaveTssShare(key secrets.TssShareKey, data interface{}) error {
	return s.saveTssShare(string(key), data)
}

func (s *Storage) saveTssShare(key string, data interface{}) error {
	switch share := data.(type) {
	case *frostTss.Config:
		return s.saveFrostShare(key, share)
	case frostTss.Config:
		return s.saveFrostShare(key, &share)
	case *keygen.LocalPartySaveData:
		return s.saveECDSAShare(key, share)
	case keygen.LocalPartySaveData:
		return s.saveECDSAShare(key, &share)
	}

	return errors.Errorf("unsupported tss share type %T", data)
}

func (s *Storage) saveECDSAShare(key string, data *keygen.LocalPartySaveData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal share data")
	}

	return s.store(key, map[string]interface{}{
		valueKey: string(raw),
	})
}

func (s *Storage) saveFrostShare(key string, data *frostTss.Config) error {
	raw, err := cbor.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal frost share data")
	}

	return s.store(key, map[string]interface{}{
		protocolKey: protocolFrost,
		encodingKey: encodingCBOR,
		valueKey:    base64.StdEncoding.EncodeToString(raw),
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

// TODO: test with resharing
func (s *Storage) GetTssShare() (interface{}, int, error) {
	shares, err := s.GetTssShares()
	if err != nil {
		return nil, -1, err
	}
	if shares.Share != nil {
		return shares.Share, tss.ProtocolID_ECDSA, nil
	}
	if shares.FrostShare != nil {
		return shares.FrostShare, tss.ProtocolID_FROST, nil
	}

	return nil, -1, errors.New("tss share not found")
}

func (s *Storage) GetTssShares() (*secrets.TssShares, error) {
	result := new(secrets.TssShares)

	ecdsaData, ok, err := s.loadOptional(keyTssShareECDSA)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ecdsa share data")
	}
	if ok {
		result.Share, err = decodeECDSAShare(ecdsaData)
		if err != nil {
			return nil, err
		}
	} else {
		legacyData, legacyOK, err := s.loadOptional(legacyKeyTssShare)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load legacy ecdsa share data")
		}
		if legacyOK {
			result.Share, err = decodeECDSAShare(legacyData)
			if err != nil {
				return nil, err
			}
		}
	}

	frostData, ok, err := s.loadOptional(keyTssShareFROST)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load frost share data")
	}
	if ok {
		result.FrostShare, err = decodeFrostShare(frostData)
		if err != nil {
			return nil, err
		}
	}

	if result.Share == nil && result.FrostShare == nil {
		return nil, errors.New("no shares found")
	}

	return result, nil
}

func decodeECDSAShare(kvData map[string]interface{}) (*keygen.LocalPartySaveData, error) {
	val, ok := kvData[valueKey].(string)
	if !ok {
		return nil, errors.New("share data not found")
	}
	data := new(keygen.LocalPartySaveData)
	if err := json.Unmarshal([]byte(val), data); err != nil {
		return nil, errors.Wrap(err, "failed to decode share data")
	}

	return data, nil
}

func decodeFrostShare(kvData map[string]interface{}) (*frostTss.Config, error) {
	val, ok := kvData[valueKey].(string)
	if !ok {
		return nil, errors.New("share data not found")
	}
	if kvData[encodingKey] != encodingCBOR {
		return nil, errors.New("unsupported frost share encoding")
	}

	raw, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode frost share data")
	}

	data := frost.EmptyConfig(curve.Secp256k1{})
	if err = cbor.Unmarshal(raw, data); err != nil {
		return nil, errors.Wrap(err, "failed to decode frost share data")
	}

	return data, nil
}

func (s *Storage) GetTemporaryTssShare() (*keygen.LocalPartySaveData, error) {
	kvData, err := s.load(keyTssShareTemp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load temporary share data")
	}

	return decodeECDSAShare(kvData)
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
