package ton

import (
	"reflect"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"gitlab.com/distributed_lab/figure/v3"
)

type RPC struct {
	IsTestnet       bool          `fig:"is_testnet,required"`
	Timeout         time.Duration `fig:"timeout,required"`
	GlobalConfigUrl string        `fig:"global_config_url,required"`
}

type Chain struct {
	Id            string `fig:"id,required"`
	Confirmations uint64 `fig:"confirmations,required"`

	Client                ton.APIClientWrapped
	BridgeContractAddress *address.Address
	RPC                   RPC
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeTON {
		panic("chain is not TON")
	}

	tonChain := Chain{
		Id:            c.Id,
		Confirmations: c.Confirmations,
	}

	err := figure.Out(&tonChain.BridgeContractAddress).
		FromInterface(c.BridgeAddresses).
		With(addrHook).
		Please()

	if err != nil {
		panic(errors.Wrap(err, "failed to obtain bridge addresses"))
	}

	err = figure.Out(&tonChain.RPC).FromInterface(c.Rpc).With(rpcHook).Please()
	if err != nil {
		panic(errors.Wrap(err, "failed to obtain TON rpc"))
	}

	return tonChain
}

var addrHook = figure.Hooks{
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

var rpcHook = figure.Hooks{
	"RPC": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case map[string]interface{}:
			var rpc RPC
			err := figure.Out(&rpc).
				FromInterface(v).
				With(figure.BaseHooks).
				Please()

			if err != nil {
				panic(errors.Wrap(err, "failed to init ton chain rpc"))
			}

			return reflect.ValueOf(rpc), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}
