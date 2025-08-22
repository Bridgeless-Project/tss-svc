package repository

import (
	"fmt"

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

func (p clientsRepository) Client(chainId string) (chain.Client, error) {
	fmt.Println("getting client for chain ID:", chainId)

	for id, cl := range p.clients {
		fmt.Printf("Available client: %s\n", id)
		fmt.Println("Client Type:", cl.ChainId())
		fmt.Println("Client Type:", cl.Type())
	}

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
