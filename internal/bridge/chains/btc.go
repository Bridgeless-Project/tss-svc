package chains

import (
	"reflect"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type BitcoinRpc struct {
	Wallet *rpcclient.Client `fig:"wallet,required"`
	Node   *rpcclient.Client `fig:"node,required"`
}
type Bitcoin struct {
	Id            string
	Confirmations uint64
	Rpc           BitcoinRpc
	Params        *chaincfg.Params
	Receivers     []btcutil.Address
}

func (c Chain) Bitcoin() Bitcoin {
	if c.Type != TypeBitcoin {
		panic("invalid chain type")
	}

	chain := Bitcoin{Id: c.Id, Confirmations: c.Confirmations}

	if err := figure.Out(&chain.Receivers).FromInterface(c.BridgeAddresses).With(figure.BaseHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to decode bitcoin receivers"))
	}
	if err := figure.Out(&chain.Rpc).FromInterface(c.Rpc).With(bitcoinHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to init bitcoin chain rpc"))
	}

	// ensuring wallet is properly configured
	_, err := chain.Rpc.Wallet.GetWalletInfo()
	if err != nil {
		panic(errors.Wrap(err, "failed to get wallet info"))
	}

	if c.Network == NetworkMainnet {
		chain.Params = &chaincfg.MainNetParams
	}
	if c.Network == NetworkTestnet {
		chain.Params = &chaincfg.TestNet3Params
	}

	return chain
}

var bitcoinHooks = figure.Hooks{
	"btcutil.Address": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case string:
			addr, err := btcutil.DecodeAddress(v, &chaincfg.MainNetParams)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to decode address")
			}

			return reflect.ValueOf(addr), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
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
