package vault

import (
	"context"
	"encoding/json"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	client "github.com/hashicorp/vault/api"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/secrets"
	"github.com/pkg/errors"
)

const (
	keyPreParams = "keygen_preparams"
	keyAccount   = "core_account"
	keyTssShare  = "tss_share"
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

	val, ok := data["value"].(string)
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
		"value": string(raw),
	})
}

func (s *Storage) SaveTssShare(data *keygen.LocalPartySaveData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal share data")
	}

	return s.store(keyTssShare, map[string]interface{}{
		"value": string(raw),
	})
}

func (s *Storage) GetCoreAccount() (*core.Account, error) {
	kvData, err := s.load(keyAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load account")
	}

	val, ok := kvData["value"].(string)
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
		"value": hexutil.Encode(account.PrivateKey().Bytes()),
	})
}
