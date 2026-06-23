package config

import (
	api "github.com/Bridgeless-Project/tss-svc/api/config"
	chain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/config"
	bridge "github.com/Bridgeless-Project/tss-svc/internal/bridge/config"
	connector "github.com/Bridgeless-Project/tss-svc/internal/core/connector/config"
	subscriber "github.com/Bridgeless-Project/tss-svc/internal/core/subscriber/config"
	p2p "github.com/Bridgeless-Project/tss-svc/internal/p2p/config"
	vault "github.com/Bridgeless-Project/tss-svc/internal/secrets/vault/config"
	tss "github.com/Bridgeless-Project/tss-svc/internal/tss/config"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/kit/pgdb"
)

type Config interface {
	comfig.Logger
	pgdb.Databaser
	vault.Secreter
	api.Listenerer
	p2p.PartiesConfigurator
	tss.SessionParamsConfigurator
	chain.Chainer
	connector.ConnectorConfigurer
	subscriber.SubscriberConfigurator
	resharing.ParamsConfigurator
	bridge.EvmSettingsConfigurator
}

type config struct {
	getter kv.Getter

	comfig.Logger
	pgdb.Databaser
	vault.Secreter
	api.Listenerer
	p2p.PartiesConfigurator
	tss.SessionParamsConfigurator
	chain.Chainer
	connector.ConnectorConfigurer
	subscriber.SubscriberConfigurator
	resharing.ParamsConfigurator
	bridge.EvmSettingsConfigurator
}

func New(getter kv.Getter) Config {
	secreter := vault.NewSecreter()

	return &config{
		getter:                    getter,
		Secreter:                  secreter,
		Logger:                    comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Databaser:                 pgdb.NewDatabaser(getter),
		Listenerer:                api.NewListenerer(getter),
		PartiesConfigurator:       p2p.NewPartiesConfigurator(getter, secreter.SecretsStorage()),
		ParamsConfigurator:        resharing.NewParamsConfigurator(getter, secreter.SecretsStorage()),
		SessionParamsConfigurator: tss.NewSessionParamsConfigurator(getter),
		Chainer:                   chain.NewChainer(getter, secreter.SecretsStorage()),
		ConnectorConfigurer:       connector.NewConnectorConfigurer(getter),
		SubscriberConfigurator:    subscriber.NewSubscriberConfigurator(getter),
		EvmSettingsConfigurator:   bridge.NewEvmSettingsConfigurator(getter),
	}
}
