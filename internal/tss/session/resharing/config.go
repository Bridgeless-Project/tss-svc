package resharing

import (
	"crypto/tls"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	p2pconf "github.com/Bridgeless-Project/tss-svc/internal/p2p/config"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

const ParamsConfigKey = "resharing_params"

type ParamsConfigurator interface {
	ResharingParams() Params
}

type LocalPartyTlsCertificateProvider interface {
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}

type Params struct {
	Epoch     uint32    `fig:"epoch,required"`
	StartTime time.Time `fig:"start_time,required"`

	Parties        []p2p.Party // fig key: list
	Threshold      uint        `fig:"threshold,required"`
	NewParticipant bool        `fig:"new_participant,required"`
}

func NewParamsConfigurator(getter kv.Getter, tslCertProvider LocalPartyTlsCertificateProvider) ParamsConfigurator {
	return &paramsConfigurator{getter: getter, tlsCertProvider: tslCertProvider}
}

type paramsConfigurator struct {
	getter          kv.Getter
	once            comfig.Once
	tlsCertProvider LocalPartyTlsCertificateProvider
}

func (p *paramsConfigurator) ResharingParams() Params {
	return p.once.Do(func() interface{} {
		var cfg Params

		err := figure.
			Out(&cfg).
			From(kv.MustGetStringMap(p.getter, ParamsConfigKey)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to figure out resharing params"))
		}

		partiesConfigurer := p2pconf.NewPartiesConfigurator(p.getter, p.tlsCertProvider, ParamsConfigKey)
		cfg.Parties = partiesConfigurer.Parties()

		return cfg
	}).(Params)
}
