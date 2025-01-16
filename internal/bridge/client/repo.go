package client

import (
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/client/btc"
	"github.com/hyle-team/tss-svc/internal/bridge/client/evm"
	"github.com/hyle-team/tss-svc/internal/bridge/client/zano"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type proxiesRepository struct {
	proxies map[string]bridgeTypes.Proxy
}

func NewProxiesRepository(chains []chain.Chain, logger *logan.Entry) (bridgeTypes.ProxiesRepository, error) {
	proxiesMap := make(map[string]bridgeTypes.Proxy, len(chains))

	for _, ch := range chains {
		var proxy bridgeTypes.Proxy

		switch ch.Type {
		case bridgeTypes.ChainTypeEVM:
			proxy = evm.NewBridgeProxy(ch.Evm(), logger)
		case bridgeTypes.ChainTypeBitcoin:
			proxy = btc.NewBridgeProxy(ch.Bitcoin(), logger)
		case bridgeTypes.ChainTypeZano:
			proxy = zano.NewBridgeProxy(ch.Zano(), logger)
		default:
			return nil, errors.Errorf("unknown chain type %s", ch.Type)
		}

		proxiesMap[ch.Id] = proxy
	}

	return &proxiesRepository{proxies: proxiesMap}, nil
}

func (p proxiesRepository) Proxy(chainId string) (bridgeTypes.Proxy, error) {
	proxy, ok := p.proxies[chainId]
	if !ok {
		return nil, bridgeTypes.ErrChainNotSupported
	}

	return proxy, nil
}

func (p proxiesRepository) SupportsChain(chainId string) bool {
	_, ok := p.proxies[chainId]
	return ok
}
