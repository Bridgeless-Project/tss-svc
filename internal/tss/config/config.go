package config

import (
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const paramsConfigKey = "tss"

type SessionParamsConfigurator interface {
	TssSessionParams() session.Params
}

type configurator struct {
	getter kv.Getter
	once   comfig.Once
}

func NewSessionParamsConfigurator(getter kv.Getter) SessionParamsConfigurator {
	return &configurator{getter: getter}
}

func (t *configurator) TssSessionParams() session.Params {
	return t.once.Do(func() interface{} {
		var params session.Params

		err := figure.
			Out(&params).
			With(figure.BaseHooks).
			From(kv.MustGetStringMap(t.getter, paramsConfigKey)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load tss params config"))
		}

		return params
	}).(session.Params)
}
