package client

import (
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/client/zano"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type clientsRepository struct {
	clients map[string]bridgeTypes.Client
}

func NewclientsRepository(chains []chain.Chain, logger *logan.Entry) (bridgeTypes.ClientsRepository, error) {
	clientsMap := make(map[string]bridgeTypes.Client, len(chains))

	for _, ch := range chains {
		var Client bridgeTypes.Client

		switch ch.Type {
		case chain.TypeEVM:
			Client = evm.NewBridgeClient(ch.Evm(), logger)
		//TODO: Add Bitcoin implementation
		case chain.TypeZano:
			Client = zano.NewBridgeClient(ch.Zano(), logger)
		default:
			return nil, errors.Errorf("unknown chain type %s", ch.Type)
		}

		clientsMap[ch.Id] = Client
	}

	return &clientsRepository{clients: clientsMap}, nil
}

func (p clientsRepository) Client(chainId string) (bridgeTypes.Client, error) {
	Client, ok := p.clients[chainId]
	if !ok {
		return nil, bridgeTypes.ErrChainNotSupported
	}

	return Client, nil
}

func (p clientsRepository) SupportsChain(chainId string) bool {
	_, ok := p.clients[chainId]
	return ok
}
