package ton

import (
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"gitlab.com/distributed_lab/figure/v3"
	"reflect"
)

type RPC struct {
	IsTestnet       bool   `fig:"is_testnet,required"`
	Timeout         uint64 `fig:"timeout,required"`
	GlobalConfigUrl string `fig:"global_config_url,required"`
}

type Chain struct {
	Id            string
	Confirmations uint64

	Client                ton.APIClientWrapped
	BridgeContractAddress *address.Address
	RPC                   RPC
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeTON {
		panic("chain is not TON")
	}

	chain := Chain{
		Id:            c.Id,
		Confirmations: c.Confirmations}

	if err := figure.Out(&chain.BridgeContractAddress).FromInterface(c.BridgeAddresses).With(addrHook()).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain bridge addresses"))
	}

	if err := figure.Out(&chain.RPC).FromInterface(c.Rpc).With(rpcHook()).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain TON rpc"))
	}

	return chain
}

func addrHook() figure.Hooks {
	return figure.Hooks{
		"*address.Address": func(value interface{}) (reflect.Value, error) {
			switch v := value.(type) {
			case string:
				addr, err := address.ParseAddr(v)
				if err != nil {
					return reflect.Value{}, errors.Wrap(err, "failed to decode address")
				}

				return reflect.ValueOf(addr), nil
			default:
				return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
			}
		},
	}
}

func rpcHook() figure.Hooks {
	return figure.Hooks{
		"RPC": func(value interface{}) (reflect.Value, error) {
			switch v := value.(type) {
			case map[string]interface{}:
				var rpc RPC
				if err := figure.Out(&rpc).FromInterface(v).With(figure.BaseHooks).Please(); err != nil {
					panic(errors.Wrap(err, "failed to init ton chain rpc"))
				}

				return reflect.ValueOf(rpc), nil
			default:
				return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
			}
		},
	}
}
