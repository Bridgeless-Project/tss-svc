package client

import (
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/client/zano"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"

	"github.com/pkg/errors"
)

type repository struct {
	clients map[string]bridgeTypes.Client
}

func NewRepository(chains []chain.Chain) (bridgeTypes.ClientsRepository, error) {
	clientsMap := make(map[string]bridgeTypes.Client, len(chains))

	for _, ch := range chains {
		var client bridgeTypes.Client

		switch ch.Type {
		case chain.TypeEVM:
			client = evm.NewBridgeClient(ch.Evm())
		//TODO: Add Bitcoin implementation
		case chain.TypeZano:
			client = zano.NewBridgeClient(ch.Zano())
		default:
			return nil, errors.Errorf("unknown chain type %s", ch.Type)
		}

		clientsMap[ch.Id] = client
	}

	return &repository{clients: clientsMap}, nil
}

func (p repository) Client(chainId string) (bridgeTypes.Client, error) {
	client, ok := p.clients[chainId]
	if !ok {
		return nil, bridgeTypes.ErrChainNotSupported
	}

	return client, nil
}

func (p repository) SupportsChain(chainId string) bool {
	_, ok := p.clients[chainId]
	return ok
}
