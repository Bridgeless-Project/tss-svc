package config

import (
	chain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/config"
	connector "github.com/Bridgeless-Project/tss-svc/internal/core/connector/config"
	subscriber "github.com/Bridgeless-Project/tss-svc/internal/core/subscriber/config"
	p2p "github.com/Bridgeless-Project/tss-svc/internal/p2p/config"
	vault "github.com/Bridgeless-Project/tss-svc/internal/secrets/vault/config"
	tss "github.com/Bridgeless-Project/tss-svc/internal/tss/config"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/kit/pgdb"
)

type Config interface {
	comfig.Logger
	pgdb.Databaser
	vault.Secreter
	Listenerer
	p2p.PartiesConfigurator
	tss.SessionParamsConfigurator
	chain.Chainer
	connector.ConnectorConfigurer
	subscriber.SubscriberConfigurator
}

type config struct {
	getter kv.Getter

	comfig.Logger
	pgdb.Databaser
	vault.Secreter
	Listenerer
	p2p.PartiesConfigurator
	tss.SessionParamsConfigurator
	chain.Chainer
	connector.ConnectorConfigurer
	subscriber.SubscriberConfigurator
}

func New(getter kv.Getter) Config {
	secreter := vault.NewSecreter()

	return &config{
		getter:                    getter,
		Secreter:                  secreter,
		Logger:                    comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Databaser:                 pgdb.NewDatabaser(getter),
		Listenerer:                NewListenerer(getter),
		PartiesConfigurator:       p2p.NewPartiesConfigurator(getter, secreter.SecretsStorage()),
		SessionParamsConfigurator: tss.NewSessionParamsConfigurator(getter),
		Chainer:                   chain.NewChainer(getter),
		ConnectorConfigurer:       connector.NewConnectorConfigurer(getter),
		SubscriberConfigurator:    subscriber.NewSubscriberConfigurator(getter),
	}
}
