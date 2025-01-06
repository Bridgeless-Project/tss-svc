package vault

import (
	"context"
	"encoding/json"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	client "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

const (
	keyPreParams = "preparams"
)

type Storage struct {
	client *client.KVv2
}

func NewStorage(client *client.KVv2) *Storage {
	return &Storage{
		client: client,
	}
}

func (s *Storage) GetKeygenPreParams() (*keygen.LocalPreParams, error) {
	kvData, err := s.client.Get(context.Background(), keyPreParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load preparams")
	}
	if kvData == nil {
		return nil, errors.New("preparams not found")
	}

	val, ok := kvData.Data["value"].(string)
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

	_, err = s.client.Put(context.Background(), keyPreParams, map[string]interface{}{
		"value": string(raw),
	})
	if err != nil {
		return errors.Wrap(err, "failed to save preparams")
	}

	return nil
}
