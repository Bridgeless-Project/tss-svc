package config

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type Swap interface {
	Contract() string
	ChainId() string
	TokenId() uint64
}

const (
	swapKey = "swap"
)

type swapsConfig struct {
	Contract string `fig:"contract,required"`
	ChainId  string `fig:"chain_id,required"`
	TokenId  uint64 `fig:"token_id,required"`
}

type swap struct {
	getter kv.Getter
	once   comfig.Once
}

func NewSwap(getter kv.Getter) Swap {
	return &swap{getter: getter}
}

func (s *swap) Contract() string {
	return s.getSwapConfig().Contract
}

func (s *swap) ChainId() string {
	return s.getSwapConfig().ChainId
}

func (s *swap) TokenId() uint64 {
	return s.getSwapConfig().TokenId
}

func (s *swap) getSwapConfig() swapsConfig {
	return s.once.Do(func() any {
		var cfg swapsConfig
		if err := figure.
			Out(&cfg).
			With(figure.BaseHooks).
			From(kv.MustGetStringMap(s.getter, swapKey)).
			Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out swap config"))
		}
		return cfg
	}).(swapsConfig)
}
