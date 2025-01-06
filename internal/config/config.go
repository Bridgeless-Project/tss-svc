package config

import (
	vaulter "github.com/hyle-team/tss-svc/internal/secrets/vault/config"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/kit/pgdb"
)

type Config interface {
	comfig.Logger
	pgdb.Databaser
	vaulter.Vaulter
}

type config struct {
	getter kv.Getter

	comfig.Logger
	pgdb.Databaser
	vaulter.Vaulter
}

func New(getter kv.Getter) Config {
	return &config{
		getter:    getter,
		Vaulter:   vaulter.NewVaulter(),
		Logger:    comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Databaser: pgdb.NewDatabaser(getter),
	}
}
