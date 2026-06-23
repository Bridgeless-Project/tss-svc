package bridge

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type EvmSettingsConfigurator interface {
	EvmSettings() EvmSettings
}

const settingsKey = "bridge_evm_config"

type EvmSettings struct {
	SwapContract common.Address `fig:"swap_contract_address,required"`
	ChainId      string         `fig:"chain_id,required"`
}

func (s EvmSettings) ChainIdAsBigInt() (*big.Int, error) {
	chainId, ok := new(big.Int).SetString(s.ChainId, 10)
	if !ok {
		return nil, errors.Errorf("invalid bridge EVM chain id: %s", s.ChainId)
	}

	return chainId, nil
}

type settinger struct {
	getter kv.Getter
	once   comfig.Once
}

func NewEvmSettingsConfigurator(getter kv.Getter) EvmSettingsConfigurator {
	return &settinger{getter: getter}
}

func (s *settinger) EvmSettings() EvmSettings {
	return s.once.Do(func() any {
		var cfg EvmSettings
		if err := figure.
			Out(&cfg).
			With(figure.BaseHooks, figure.EthereumHooks).
			From(kv.MustGetStringMap(s.getter, settingsKey)).
			Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out bridge EVM config"))
		}
		return cfg
	}).(EvmSettings)
}
