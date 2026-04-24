package evm

import (
	"crypto/ecdsa"
	"reflect"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type Chain struct {
	Id            string
	Rpc           *ethclient.Client
	RawClient     *rpc.Client
	BridgeAddress common.Address
	Confirmations uint64

	Meta Meta
}

type Meta struct {
	Standart    bool              `fig:"standart"`
	Centralized bool              `fig:"centralized"`
	SignerKey   *ecdsa.PrivateKey `fig:"signer_key"`
}

func (m *Meta) ValidateE() error {
	if m.Centralized && m.SignerKey == nil {
		return errors.New("signer_key is required for centralized EVM chains")
	}

	return nil
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeEVM {
		panic("chain is not EVM")
	}

	chain := Chain{
		Id:            c.Id,
		Confirmations: c.Confirmations,
	}

	if err := figure.Out(&chain.Rpc).FromInterface(c.Rpc).With(figure.EthereumHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain Ethereum clients"))
	}
	if err := figure.Out(&chain.RawClient).FromInterface(c.Rpc).With(RawClientHook).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain raw Ethereum RPC client"))
	}
	if err := figure.Out(&chain.BridgeAddress).FromInterface(c.BridgeAddresses).With(figure.EthereumHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain bridge addresses"))
	}
	if err := figure.Out(&chain.Meta).FromInterface(c.Meta).With(figure.BaseHooks, figure.EthereumHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to decode chain meta"))
	}
	if err := chain.Meta.ValidateE(); err != nil {
		panic(errors.Wrap(err, "invalid chain meta"))
	}

	return chain
}

var RawClientHook = figure.Hooks{
	"*rpc.Client": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case string:
			client, err := rpc.Dial(v)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(client), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}
