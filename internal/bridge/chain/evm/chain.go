package evm

import (
	"crypto/ecdsa"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type Chain struct {
	Id            string
	Rpc           *ethclient.Client
	BridgeAddress common.Address
	Confirmations uint64

	Meta Meta
}

type Meta struct {
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
	if err := figure.Out(&chain.BridgeAddress).FromInterface(c.BridgeAddresses).With(figure.EthereumHooks).Please(); err != nil {
		panic(errors.Wrap(err, "failed to obtain bridge addresses"))
	}
	if err := figure.Out(&chain.Meta).FromInterface(c.Meta).Please(); err != nil {
		panic(errors.Wrap(err, "failed to decode chain meta"))
	}
	if err := chain.Meta.ValidateE(); err != nil {
		panic(errors.Wrap(err, "invalid chain meta"))
	}

	return chain
}
