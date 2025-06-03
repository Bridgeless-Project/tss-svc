package config

import (
	"github.com/hyle-team/tss-svc/internal/bridge/chain/ton"
	"reflect"

	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/bitcoin"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/zano"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type Chainer interface {
	Chains() []chain.Chain
	Clients() []chain.Client
}

type chainer struct {
	chainsOnce  comfig.Once
	clientsOnce comfig.Once
	getter      kv.Getter
}

func NewChainer(getter kv.Getter) Chainer {
	return &chainer{
		getter: getter,
	}
}

func (c *chainer) Clients() []chain.Client {
	return c.clientsOnce.Do(func() interface{} {
		chains := c.Chains()
		clients := make([]chain.Client, len(chains))

		for i, ch := range chains {
			switch ch.Type {
			case chain.TypeZano:
				clients[i] = zano.NewBridgeClient(zano.FromChain(ch))
			case chain.TypeEVM:
				clients[i] = evm.NewBridgeClient(evm.FromChain(ch))
			case chain.TypeBitcoin:
				clients[i] = bitcoin.NewBridgeClient(bitcoin.FromChain(ch))
			case chain.TypeTON:
				clients[i] = ton.NewBridgeClient(ton.FromChain(ch))
			default:
				panic(errors.Errorf("unsupported chain type: %s", ch.Type))
			}
		}

		return clients
	}).([]chain.Client)
}

func (c *chainer) Chains() []chain.Chain {
	return c.chainsOnce.Do(func() interface{} {
		var cfg struct {
			Chains []chain.Chain `fig:"list,required"`
		}

		if err := figure.
			Out(&cfg).
			With(
				figure.BaseHooks,
				figure.EthereumHooks,
				interfaceHook,
			).
			From(kv.MustGetStringMap(c.getter, "chains")).
			Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out chain"))
		}

		if len(cfg.Chains) == 0 {
			panic(errors.New("no chain were configured"))
		}

		return cfg.Chains
	}).([]chain.Chain)
}

// simple hook to delay parsing interface details
var interfaceHook = figure.Hooks{
	"interface {}": func(value interface{}) (reflect.Value, error) {
		return reflect.ValueOf(value), nil
	},
}
