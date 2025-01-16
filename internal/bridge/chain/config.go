package chain

import (
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"reflect"
)

type Chainer interface {
	Chains() []Chain
}

type chainer struct {
	once   comfig.Once
	getter kv.Getter
}

func NewChainer(getter kv.Getter) Chainer {
	return &chainer{
		getter: getter,
	}
}

func (c *chainer) Chains() []Chain {
	return c.once.Do(func() interface{} {
		var cfg struct {
			Chains []Chain `fig:"list,required"`
		}

		if err := figure.
			Out(&cfg).
			With(
				figure.BaseHooks,
				figure.EthereumHooks,
				bitcoinHooks,
				interfaceHook,
			).
			From(kv.MustGetStringMap(c.getter, "chains")).
			Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out chains"))
		}

		if len(cfg.Chains) == 0 {
			panic(errors.New("no chains were configured"))
		}

		return cfg.Chains
	}).([]Chain)
}

// simple hook to delay parsing interface details
var interfaceHook = figure.Hooks{
	"interface {}": func(value interface{}) (reflect.Value, error) {
		return reflect.ValueOf(value), nil
	},
}

type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
)

func (n Network) Validate() error {
	switch n {
	case NetworkMainnet, NetworkTestnet:
		return nil
	default:
		return errors.New("invalid network")
	}
}

type Chain struct {
	Id              string                `fig:"id,required"`
	Type            bridgeTypes.ChainType `fig:"type,required"`
	Confirmations   uint64                `fig:"confirmations,required"`
	Rpc             any                   `fig:"rpc,required"`
	BridgeAddresses any                   `fig:"bridge_addresses,required"`

	Wallet  string  `fig:"wallet"`
	Network Network `fig:"network"`
}
