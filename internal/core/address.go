package core

import (
	"fmt"
	"reflect"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

type Address string

func AddressFromString(s string) (Address, error) {
	addr := Address(s)

	if addr.Validate() != nil {
		return "", errors.New("invalid address")
	}

	return addr, nil
}

func (a Address) String() string {
	return string(a)
}

func (a Address) Validate() error {
	_, _, err := bech32.DecodeAndConvert(a.String())

	return errors.Wrap(err, "failed to decode address")
}

func (a Address) Bytes() []byte {
	_, data, err := bech32.DecodeAndConvert(a.String())
	if err != nil {
		panic(err)
	}

	return data
}

var AddressHook = figure.Hooks{
	"core.Address": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case string:
			addr, err := AddressFromString(v)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to unmarshal address")
			}

			return reflect.ValueOf(addr), nil
		default:
			return reflect.Value{}, fmt.Errorf("unexpected type %T for core.Address", value)
		}
	},
}
