package repository

import (
	"github.com/hyle-team/tss-svc/internal/bridge/chains"
	"github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/bitcoin"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/zano"
)

type clientsRepository struct {
	clients map[string]clients.Client
}

func NewClientsRepository(chs []chains.Chain) clients.Repository {
	clientsMap := make(map[string]clients.Client, len(chs))

	for _, ch := range chs {
		var cl clients.Client

		switch ch.Type {
		case chains.TypeEVM:
			cl = evm.NewBridgeClient(ch.Evm())
		case chains.TypeBitcoin:
			cl = bitcoin.NewBridgeClient(ch.Bitcoin())
		case chains.TypeZano:
			cl = zano.NewBridgeClient(ch.Zano())
		default:
			continue
		}

		clientsMap[ch.Id] = cl
	}

	return &clientsRepository{clients: clientsMap}
}

func (p clientsRepository) Client(chainId string) (clients.Client, error) {
	cl, ok := p.clients[chainId]
	if !ok {
		return nil, clients.ErrChainNotSupported
	}

	return cl, nil
}

func (p clientsRepository) SupportsChain(chainId string) bool {
	_, ok := p.clients[chainId]
	return ok
}
