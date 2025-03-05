package config

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type SyncerConfig struct {
	MaxRetries int `fig:"max_retries,required"`
}

const (
	syncerConfigKey = "syncer"
)

type syncConfigurator struct {
	getter kv.Getter
	once   comfig.Once
}

func NewSyncConfigurator(getter kv.Getter) SyncerConfigurator {
	return &syncConfigurator{getter: getter}
}

type SyncerConfigurator interface {
	MaxRetries() int
}

func (s *syncConfigurator) MaxRetries() int {
	return s.once.Do(func() interface{} {
		var cfg SyncerConfig

		err := figure.Out(&cfg).
			From(kv.MustGetStringMap(s.getter, syncerConfigKey)).
			With(figure.BaseHooks).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load syncer config"))
		}

		return cfg.MaxRetries
	}).(int)
}
