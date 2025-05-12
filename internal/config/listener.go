package config

import (
	"net"
	"reflect"

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
	P2pGrpc net.Listener `fig:"p2p_grpc_addr,required"`
	ApiGrpc net.Listener `fig:"api_grpc_addr,required"`
	ApiHttp net.Listener `fig:"api_http_addr,required"`
}

type listener struct {
	getter kv.Getter
	once   comfig.Once
}

func (l *listener) P2pGrpcListener() net.Listener {
	return l.listener(listenersKey).P2pGrpc
}

func (l *listener) ApiGrpcListener() net.Listener {
	return l.listener(listenersKey).ApiGrpc
}

func (l *listener) ApiHttpListener() net.Listener {
	return l.listener(listenersKey).ApiHttp
}

func (l *listener) listener(key string) listeners {
	return l.once.Do(func() interface{} {
		var ls listeners
		err := figure.
			Out(&ls).
			With(figure.BaseHooks, listenerHooks).
			From(kv.MustGetStringMap(l.getter, key)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load listener config"))
		}

		return ls
	}).(listeners)
}

var listenerHooks = figure.Hooks{
	"net.Listener": func(value interface{}) (reflect.Value, error) {
		switch addr := value.(type) {
		case string:
			ls, err := net.Listen("tcp", addr)
			if err != nil {
				return reflect.Value{}, errors.Wrapf(err, "failed to listen on %s", addr)
			}

			return reflect.ValueOf(ls), nil
		default:
			return reflect.Value{}, errors.Errorf("unsupported conversion from %T", value)
		}
	},
}
