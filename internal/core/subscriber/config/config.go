package config

import (
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const subscriberConfigKey = "subscriber"

type SubscriberConfigurator interface {
	TendermintHttpClient() *http.HTTP
}

type subscriber struct {
	once   comfig.Once
	getter kv.Getter
}

func NewSubscriberConfigurator(getter kv.Getter) SubscriberConfigurator {
	return &subscriber{
		getter: getter,
	}
}

func (sc *subscriber) TendermintHttpClient() *http.HTTP {
	return sc.once.Do(func() interface{} {
		var config struct {
			Addr string `fig:"addr"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(sc.getter, subscriberConfigKey)).Please(); err != nil {
			panic(errors.Wrap(err, "failed to figure out core subscriber config"))
		}

		client, err := http.New(config.Addr, "/websocket")
		if err != nil {
			panic(errors.Wrap(err, "failed to create tendermint http client"))
		}

		if err = client.Start(); err != nil {
			panic(errors.Wrap(err, "failed to start tendermint http client"))
		}

		return client
	}).(*http.HTTP)
}
