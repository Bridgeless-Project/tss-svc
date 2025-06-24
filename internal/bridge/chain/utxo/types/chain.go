package types

import (
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

const DefaultChain = ChainBtc

var _ figure.Validatable = Chain("")

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
