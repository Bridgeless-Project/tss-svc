package types

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

var _, _ figure.Validatable = Network(""), Type("")

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

type Type string

const (
	TypeBtc Type = "btc"
	TypeBch Type = "bch"
)

func (s Type) Validate() error {
	switch s {
	case TypeBtc, TypeBch:
		return nil
	default:
		return errors.Errorf("invalid type: %s", s)
	}
}
