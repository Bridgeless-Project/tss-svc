package chain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type EvmChain struct {
	Id            string
	Rpc           *ethclient.Client
	BridgeAddress common.Address
	Confirmations uint64
}

func (c Chain) Evm() EvmChain {
	if c.Type != TypeEVM {
		panic("chain is not EVM")
	}

	chain := EvmChain{
		Id:            c.Id,
		Confirmations: c.Confirmations}

	if err := figure.Out(&chain.Rpc).FromInterface(c.Rpc).With(figure.EthereumHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain Ethereum client"))
	}
	if err := figure.Out(&chain.BridgeAddress).FromInterface(c.BridgeAddresses).With(figure.EthereumHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain bridge addresses"))
	}

	return chain
}
