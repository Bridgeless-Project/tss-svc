package chain

import (
	"reflect"

	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/rpc"
	utxotypes "github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type Chain struct {
	Id            string
	Confirmations uint64
	Rpc           Rpc
	Receivers     []string

	Meta Meta
}

type Rpc struct {
	Wallet *rpc.Client `fig:"wallet,required"`
	Node   *rpc.Client `fig:"node,required"`
}

type Meta struct {
	Network utxotypes.Network
	Type    utxotypes.Type
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeBitcoin {
		panic("invalid chain type")
	}

	ch := Chain{
		Id:            c.Id,
		Confirmations: c.Confirmations,
	}

	if err := figure.Out(&ch.Meta).FromInterface(c.Meta).Please(); err != nil {
		panic(errors.Wrap(err, "failed to decode chain meta"))
	}
	if err := figure.Out(&ch.Rpc).FromInterface(c.Rpc).With(clientHook).Please(); err != nil {
		panic(errors.Wrap(err, "failed to init bitcoin chain rpc"))
	}
	if err := figure.Out(&ch.Receivers).FromInterface(c.BridgeAddresses).Please(); err != nil {
		panic(errors.Wrap(err, "failed to decode bitcoin receivers"))
	}

	helper := helper.NewUtxoHelper(ch.Meta.Type, ch.Meta.Network)
	for _, addr := range ch.Receivers {
		if !helper.AddressValid(addr) {
			panic(errors.Errorf("invalid receiver address: %s", addr))
		}
	}

	// ensuring wallet is properly configured
	if _, err := ch.Rpc.Wallet.GetWalletInfo(); err != nil {
		panic(errors.Wrap(err, "failed to get wallet info"))
	}

	return ch
}

var clientHook = figure.Hooks{
	"*rpc.Client": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case map[string]interface{}:
			var clientConfig struct {
				Host string `fig:"host,required"`
				User string `fig:"user,required"`
				Pass string `fig:"pass,required"`
			}

			if err := figure.Out(&clientConfig).With(figure.BaseHooks).From(v).Please(); err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to figure out bitcoin rpc client config")
			}

			client, err := rpc.NewRpcClient(
				clientConfig.Host,
				clientConfig.User,
				clientConfig.Pass,
			)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to create bitcoin rpc client")
			}

			return reflect.ValueOf(client), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}
