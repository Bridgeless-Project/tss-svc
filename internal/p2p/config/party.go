package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"reflect"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	partiesConfigKey = "parties"
)

type PartiesConfigurator interface {
	Parties() []p2p.Party
}

type LocalPartyTlsCertificateProvider interface {
	GetLocalPartyTlsCertificate() (*tls.Certificate, error)
}

type Party struct {
	CoreAddress        core.Address `fig:"core_address,required"`
	Connection         string       `fig:"connection,required"`
	TlsCertificatePath string       `fig:"tls_certificate_path,required"`
}

func NewPartiesConfigurator(getter kv.Getter, tslCertProvider LocalPartyTlsCertificateProvider) PartiesConfigurator {
	return &partiesConfigurator{getter: getter, tlsCertProvider: tslCertProvider}
}

type partiesConfigurator struct {
	getter          kv.Getter
	once            comfig.Once
	tlsCertProvider LocalPartyTlsCertificateProvider
}

func (p *partiesConfigurator) Parties() []p2p.Party {
	return p.once.Do(func() interface{} {
		var cfg struct {
			Parties []p2p.Party `fig:"list,required"`
		}

		err := figure.
			Out(&cfg).
			From(kv.MustGetStringMap(p.getter, partiesConfigKey)).
			With(figure.BaseHooks, partyHook(p.tlsCertProvider)).
			Please()
		if err != nil {
			panic(errors.Wrap(err, "failed to load parties config"))
		}

		return cfg.Parties
	}).([]p2p.Party)
}

func partyHook(localCertProvider LocalPartyTlsCertificateProvider) figure.Hooks {
	return figure.Hooks{
		"p2p.Party": func(value interface{}) (reflect.Value, error) {
			raw, ok := value.(map[string]interface{})
			if !ok {
				return reflect.Value{}, fmt.Errorf("unexpected type %T", value)
			}

			var partyConf Party
			if err := figure.Out(&partyConf).From(raw).With(figure.BaseHooks, core.AddressHook).Please(); err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to unmarshal party")
			}

			partyCert, err := os.ReadFile(partyConf.TlsCertificatePath)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to read party certificate")
			}

			conn, err := configureConnection(partyConf, partyCert, localCertProvider)
			if err != nil {
				return reflect.Value{}, errors.Wrap(err, "failed to connect party")
			}

			return reflect.ValueOf(p2p.NewParty(partyConf.CoreAddress, conn, partyCert)), nil

		},
	}
}

func configureConnection(party Party, partyCertPEM []byte, certProvider LocalPartyTlsCertificateProvider) (*grpc.ClientConn, error) {
	selfCert, err := certProvider.GetLocalPartyTlsCertificate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get local party tls certificate")
	}

	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(partyCertPEM) {
		return nil, errors.New("invalid party certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*selfCert},
		RootCAs:      rootCAs,
	}
	creds := credentials.NewTLS(tlsConfig)

	return grpc.NewClient(party.Connection, grpc.WithTransportCredentials(creds))
}
