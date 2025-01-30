package config

import (
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type SubscriberConfig interface {
	Client() *http.HTTP
}

type subscriber struct {
	once   comfig.Once
	getter kv.Getter
}

func NewSubscriberConfig(getter kv.Getter) SubscriberConfig {
	return &subscriber{
		getter: getter,
	}
}

func (sc *subscriber) Client() *http.HTTP {
	return sc.once.Do(func() interface{} {
		var config struct {
			Addr string `fig:"addr"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(sc.getter, "subscriber")).Please(); err != nil {
			panic(err)
		}

		client, err := http.New(config.Addr, "/websocket")
		if err != nil {
			panic(err)
		}

		if err := client.Start(); err != nil {
			panic(err)
		}

		return client
	}).(*http.HTTP)
}
