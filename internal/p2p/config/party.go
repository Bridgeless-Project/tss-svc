package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	partiesConfigKey = "parties"
)

type PartiesConfigurator interface {
	Parties() []p2p.Party
}

type Party struct {
	PubKey      string       `fig:"pubkey,required"`
	CoreAddress core.Address `fig:"core_address,required"`
	Connection  string       `fig:"connection,required"`
}

func NewPartiesConfigurator(getter kv.Getter) PartiesConfigurator {
	return &partiesConfigurator{getter: getter}
}

type partiesConfigurator struct {
	getter kv.Getter
	once   comfig.Once
}

func (p *partiesConfigurator) Parties() []p2p.Party {
	return p.once.Do(func() interface{} {
		var cfg struct {
			Parties []p2p.Party `fig:"list,required"`
		}

		err := figure.
			Out(&cfg).
			From(kv.MustGetStringMap(p.getter, partiesConfigKey)).
			With(figure.BaseHooks, partyHook).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load parties config"))
		}

		return cfg.Parties
	}).([]p2p.Party)
}

var partyHook = figure.Hooks{
	"p2p.Party": func(value interface{}) (reflect.Value, error) {
		switch v := value.(type) {
		case map[string]interface{}:
			var raw Party

			if err := figure.Out(&raw).From(v).With(figure.BaseHooks, core.AddressHook).Please(); err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to unmarshal party")
			}

			conn, err := connect(raw)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to connect party")
			}

			return reflect.ValueOf(p2p.NewParty(raw.PubKey, raw.CoreAddress, conn)), nil
		default:
			return reflect.Value{}, fmt.Errorf("unexpected type %T", value)
		}
	},
}

// TODO: expand with mTLS
func connect(party Party) (*grpc.ClientConn, error) {
	insecureOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	keepaliveOpt := grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:    20 * time.Second, // wait time before ping if no activity
		Timeout: 5 * time.Second,  // ping timeout
	})

	return grpc.NewClient(party.Connection, insecureOpt, keepaliveOpt)
}
