package types

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

const (
	DefaultNetwork = NetworkMainnet
	DefaultType    = ChainBtc
)

var _, _ figure.Validatable = Network(""), Chain("")

type Network string

const (
	NetworkMainnet  Network = "mainnet"
	NetworkTestnet3 Network = "testnet3"
	NetworkTestnet4 Network = "testnet4"
)

func (n Network) Validate() error {
	switch n {
	case NetworkMainnet, NetworkTestnet3, NetworkTestnet4:
		return nil
	default:
		return errors.Errorf("invalid network: %s", n)
	}
}

type Chain string

const (
	ChainBtc Chain = "btc"
	ChainBch Chain = "bch"
)

func (s Chain) Validate() error {
	switch s {
	case ChainBtc, ChainBch:
		return nil
	default:
		return errors.Errorf("invalid type: %s", s)
	}
}
