package config

import (
	"cmp"
	"os"

	vault "github.com/hashicorp/vault/api"
	"gitlab.com/distributed_lab/kit/comfig"
)

const (
	VaultPathEnv   = "VAULT_PATH"
	VaultTokenEnv  = "VAULT_TOKEN"
	VaultMountPath = "MOUNT_PATH"
)

type Vaulter interface {
	VaultClient() *vault.KVv2
}

type vaulter struct {
	once comfig.Once
}

func NewVaulter() Vaulter {
	return &vaulter{}
}

func (v *vaulter) VaultClient() *vault.KVv2 {
	return v.once.Do(func() interface{} {
		conf := vault.DefaultConfig()
		conf.Address = os.Getenv(VaultPathEnv)

		client, err := vault.NewClient(conf)
		if err != nil {
			panic(err)
		}

		client.SetToken(os.Getenv(VaultTokenEnv))

		mountPath := cmp.Or(os.Getenv(VaultMountPath), "secret")
		return client.KVv2(mountPath)
	}).(*vault.KVv2)
}
