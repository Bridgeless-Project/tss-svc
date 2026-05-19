package bridge

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type SwapConfigurator interface {
	SwapSettings() SwapSettings
}

const swapKey = "swap_config"

type SwapSettings struct {
	Contract      string `fig:"contract_address"`
	ChainId       string `fig:"chain_id"`
	WrappedBridge uint64 `fig:"wrapped_bridge"`
}

type swapper struct {
	getter kv.Getter
	once   comfig.Once
}

func NewSwapConfigurator(getter kv.Getter) SwapConfigurator {
	return &swapper{getter: getter}
}

func (s *swapper) SwapSettings() SwapSettings {
	return s.once.Do(func() any {
		var cfg SwapSettings
		if err := figure.
			Out(&cfg).
			With(figure.BaseHooks).
			From(kv.MustGetStringMap(s.getter, swapKey)).
			Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out swap config"))
		}
		return cfg
	}).(SwapSettings)
}
