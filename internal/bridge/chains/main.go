package chains

import (
	"github.com/pkg/errors"
)

type Chain struct {
	Id              string `fig:"id,required"`
	Type            Type   `fig:"type,required"`
	Confirmations   uint64 `fig:"confirmations,required"`
	Rpc             any    `fig:"rpc,required"`
	BridgeAddresses any    `fig:"bridge_addresses,required"`

	Wallet  string  `fig:"wallet"`
	Network Network `fig:"network"`
}

type Type string

const (
	TypeEVM   Type = "evm"
	TypeZano  Type = "zano"
	TypeOther Type = "other"
)

var typesMap = map[Type]struct{}{
	TypeEVM:   {},
	TypeZano:  {},
	TypeOther: {},
}

func (c Type) Validate() error {
	if _, ok := typesMap[c]; !ok {
		return errors.New("invalid chains type")
	}

	return nil
}

type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
)

var networksMap = map[Network]struct{}{
	NetworkMainnet: {},
	NetworkTestnet: {},
}

func (n Network) Validate() error {
	if _, ok := networksMap[n]; !ok {
		return errors.New("invalid network")
	}

	return nil
}
