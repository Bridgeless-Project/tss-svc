package config

import (
	"cmp"
	"os"

	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets/vault"
	vaultApi "github.com/hashicorp/vault/api"
	"gitlab.com/distributed_lab/kit/comfig"
)

const (
	VaultPathEnv   = "VAULT_PATH"
	VaultTokenEnv  = "VAULT_TOKEN"
	VaultMountPath = "MOUNT_PATH"
)

type Secreter interface {
	SecretsStorage() secrets.Storage
}

type vaulter struct {
	once comfig.Once
}

func NewSecreter() Secreter {
	return &vaulter{}
}

func (v *vaulter) SecretsStorage() secrets.Storage {
	return v.once.Do(func() interface{} {
		conf := vaultApi.DefaultConfig()
		conf.Address = os.Getenv(VaultPathEnv)

		client, err := vaultApi.NewClient(conf)
		if err != nil {
			panic(err)
		}

		client.SetToken(os.Getenv(VaultTokenEnv))

		mountPath := cmp.Or(os.Getenv(VaultMountPath), "secret")

		return vault.NewStorage(client.KVv2(mountPath))
	}).(secrets.Storage)
}
