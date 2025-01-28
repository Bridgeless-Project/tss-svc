package config

import (
	"time"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const paramsConfigKey = "tss"

type ParamsConfigurator interface {
	TSSParams() Params
}

type Params struct {
	Keygen  KeygenParams  `fig:"keygen"`
	Signing SigningParams `fig:"signing"`
}

func (p Params) KeygenSessionParams() session.KeygenSessionParams {
	return session.KeygenSessionParams{
		Id:        p.Keygen.Id,
		StartTime: p.Keygen.StartTime,
	}
}

func (p Params) DefaultSigningSessionParams() session.DefaultSigningSessionParams {
	return session.DefaultSigningSessionParams{
		Id:        p.Signing.Id,
		StartTime: p.Signing.StartTime,
		Threshold: p.Signing.Threshold,
	}
}

func (p Params) SigningSessionParams() session.SigningSessionParams {
	return session.SigningSessionParams{
		Id:        p.Signing.Id,
		StartTime: p.Signing.StartTime,
		Threshold: p.Signing.Threshold,
	}
}

type KeygenParams struct {
	Id        int64     `fig:"session_id,required"`
	StartTime time.Time `fig:"start_time,required"`
}

type SigningParams struct {
	Id        int64     `fig:"session_id,required"`
	StartTime time.Time `fig:"start_time,required"`
	Threshold int       `fig:"threshold,required"`
}

type tssParamsConfigurator struct {
	getter kv.Getter
	once   comfig.Once
}

func NewParamsConfigurator(getter kv.Getter) ParamsConfigurator {
	return &tssParamsConfigurator{getter: getter}
}

func (t *tssParamsConfigurator) TSSParams() Params {
	return t.once.Do(func() interface{} {
		var cfg Params

		err := figure.
			Out(&cfg).
			With(figure.BaseHooks, core.AddressHook).
			From(kv.MustGetStringMap(t.getter, paramsConfigKey)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load tss params config"))
		}

		return cfg
	}).(Params)
}
