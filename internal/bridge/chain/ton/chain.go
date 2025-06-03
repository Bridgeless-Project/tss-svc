package ton

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"gitlab.com/distributed_lab/figure/v3"
	"reflect"
)

type RPC struct {
	IsTestnet       bool   `fig:"is_testnet, required"`
	Timeout         uint64 `fig:"timeout, required"`
	GlobalConfigUrl string `fig:"global_config_url, required"`
}

type Chain struct {
	Id            string
	Confirmations uint64

	ton.APIClientWrapped
	BridgeContractAddress common.Address
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

	if err := figure.Out(&chain.RPC).FromInterface(c.Rpc).With(figure.BaseHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to init bitcoin chain rpc"))
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
