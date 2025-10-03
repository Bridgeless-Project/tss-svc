package repository

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
)

type clientsRepository struct {
	clients map[string]chain.Client
}

func NewClientsRepository(clients []chain.Client) chain.Repository {
	clientsMap := make(map[string]chain.Client, len(clients))

	for _, cl := range clients {
		clientsMap[cl.ChainId()] = cl
	}

	return &clientsRepository{clients: clientsMap}
}

func (p clientsRepository) Clients() map[string]chain.Client {
	return p.clients
}

func (p clientsRepository) Client(chainId string) (chain.Client, error) {
	cl, ok := p.clients[chainId]
	if !ok {
		return nil, chain.ErrChainNotSupported
	}

	return cl, nil
}

func (p clientsRepository) SupportsChain(chainId string) bool {
	_, ok := p.clients[chainId]
	return ok
}
