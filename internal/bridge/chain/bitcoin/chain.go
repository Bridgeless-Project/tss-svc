package bitcoin

import (
	"reflect"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type Rpc struct {
	Wallet *rpcclient.Client `fig:"wallet,required"`
	Node   *rpcclient.Client `fig:"node,required"`
}
type Chain struct {
	Id            string
	Confirmations uint64
	Rpc           Rpc
	Params        *chaincfg.Params
	Receivers     []btcutil.Address
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeBitcoin {
		panic("invalid chain type")
	}

	ch := Chain{Id: c.Id, Confirmations: c.Confirmations}
	if c.Network == chain.NetworkMainnet {
		ch.Params = &chaincfg.MainNetParams
	} else if c.Network == chain.NetworkTestnet {
		ch.Params = &chaincfg.TestNet3Params
	} else {
		panic("invalid network")
	}

	if err := figure.Out(&ch.Rpc).FromInterface(c.Rpc).With(clientHook).Please(); err != nil {
		panic(errors.Wrap(err, "failed to init bitcoin chain rpc"))
	}
	if err := figure.Out(&ch.Receivers).FromInterface(c.BridgeAddresses).With(addrHook(ch.Params)).Please(); err != nil {
		panic(errors.Wrap(err, "failed to decode bitcoin receivers"))
	}

	// ensuring wallet is properly configured
	_, err := ch.Rpc.Wallet.GetWalletInfo()
	if err != nil {
		panic(errors.Wrap(err, "failed to get wallet info"))
	}

	return ch
}

func addrHook(params *chaincfg.Params) figure.Hooks {
	return figure.Hooks{
		"btcutil.Address": func(value interface{}) (reflect.Value, error) {
			switch v := value.(type) {
			case string:
				addr, err := btcutil.DecodeAddress(v, params)
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

var clientHook = figure.Hooks{
	"*rpcclient.Client": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case map[string]interface{}:
			var clientConfig struct {
				Host       string `fig:"host,required"`
				User       string `fig:"user,required"`
				Pass       string `fig:"pass,required"`
				DisableTLS bool   `fig:"disable_tls"`
			}

			if err := figure.Out(&clientConfig).With(figure.BaseHooks).From(v).Please(); err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to figure out bitcoin rpc client config")
			}

			client, err := rpcclient.New(&rpcclient.ConnConfig{
				Host:         clientConfig.Host,
				User:         clientConfig.User,
				Pass:         clientConfig.Pass,
				HTTPPostMode: true,
				DisableTLS:   clientConfig.DisableTLS,
			}, nil)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to create bitcoin rpc client")
			}

			return reflect.ValueOf(client), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}
