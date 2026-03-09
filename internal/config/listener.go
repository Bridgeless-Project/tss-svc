package config

import (
	"net"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type Listenerer interface {
	P2pGrpcListener() net.Listener
	ApiGrpcListener() net.Listener
	ApiHttpListener() net.Listener
}

const (
	listenersKey = "listeners"
)

func NewListenerer(getter kv.Getter) Listenerer {
	return &listener{getter: getter}
}

type listeners struct {
	P2pGrpc string `fig:"p2p_grpc_addr,required"`
	ApiGrpc string `fig:"api_grpc_addr,required"`
	ApiHttp string `fig:"api_http_addr,required"`
}

type listener struct {
	getter kv.Getter
	once   comfig.Once
}

func (l *listener) P2pGrpcListener() net.Listener {
	return l.dial(l.listeners(listenersKey).P2pGrpc)
}

func (l *listener) ApiGrpcListener() net.Listener {
	return l.dial(l.listeners(listenersKey).ApiGrpc)
}

func (l *listener) ApiHttpListener() net.Listener {
	return l.dial(l.listeners(listenersKey).ApiHttp)
}

func (l *listener) listeners(key string) listeners {
	return l.once.Do(func() interface{} {
		var ls listeners
		err := figure.
			Out(&ls).
			With(figure.BaseHooks).
			From(kv.MustGetStringMap(l.getter, key)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load listener config"))
		}

		return ls
	}).(listeners)
}

func (l *listener) dial(addr string) net.Listener {
	ls, err := net.Listen("tcp", addr)
	if err != nil {
		panic(errors.Wrapf(err, "failed to listen on %s", addr))
	}

	return ls
}
