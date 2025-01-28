package zano

import (
	"regexp"

	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
)

var addressPattern = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{97}$`)

type BridgeClient interface {
	bridgeTypes.Client
	EmitAssetUnsigned(data db.Deposit) (*UnsignedTransaction, error)
	EmitAssetSigned(transaction SignedTransaction) (txHash string, err error)
}

type client struct {
	chain chain.Zano
}

func (p *client) ChainId() string {
	return p.chain.Id
}

func (p *client) Type() chain.Type {
	return chain.TypeZano
}

func (p *client) AddressValid(addr string) bool {
	return addressPattern.MatchString(addr)
}

func (p *client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func NewBridgeClient(chain chain.Zano) BridgeClient {
	return &client{chain}
}
