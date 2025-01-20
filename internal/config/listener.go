package config

import (
	"net"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type Listenerer interface {
	GRPCListener() net.Listener
	HTTPListener() net.Listener
}

const (
	grpcKey = "listener-grpc"
	httpKey = "listener-http"
)

func NewListenerer(getter kv.Getter) Listenerer {
	return &listener{getter: getter}
}

type listener struct {
	getter   kv.Getter
	grpcOnce comfig.Once
	httpOnce comfig.Once
}

func (l *listener) GRPCListener() net.Listener {
	return l.listener(&l.grpcOnce, grpcKey)
}

func (l *listener) HTTPListener() net.Listener {
	return l.listener(&l.httpOnce, httpKey)
}

func (l *listener) listener(once *comfig.Once, key string) net.Listener {
	return once.Do(func() interface{} {
		var config struct {
			Addr string `fig:"addr,required"`
		}
		err := figure.
			Out(&config).
			From(kv.MustGetStringMap(l.getter, key)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load listener config"))
		}

		ls, err := net.Listen("tcp", config.Addr)
		if err != nil {
			panic(errors.Wrap(err, "failed to bind listener address"))
		}

		return ls
	}).(net.Listener)
}
