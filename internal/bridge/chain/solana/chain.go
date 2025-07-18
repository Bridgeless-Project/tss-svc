package solana

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"reflect"
)

type Chain struct {
	Id            string
	Rpc           *rpc.Client
	BridgeAddress solana.PublicKey
	Confirmations uint64
	BridgeId      string
}

var SolanaHooks = figure.Hooks{
	"*rpc.Client": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case string:
			client := rpc.New(v)
			return reflect.ValueOf(client), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
	"solana.PublicKey": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case string:
			pubKey, err := solana.PublicKeyFromBase58(v)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(pubKey), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeSolana {
		panic("chain is not Solana")
	}
	chain := Chain{
		Id:            c.Id,
		Confirmations: c.Confirmations,
		BridgeId:      c.BridgeId,
	}

	if err := figure.Out(&chain.Rpc).FromInterface(c.Rpc).With(SolanaHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain Solana clients"))
	}
	if err := figure.Out(&chain.BridgeAddress).FromInterface(c.BridgeAddresses).With(SolanaHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain bridge addresses"))
	}

	return chain
}
