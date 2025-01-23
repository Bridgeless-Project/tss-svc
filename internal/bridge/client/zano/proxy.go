package zano

import (
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"

	"gitlab.com/distributed_lab/logan/v3"
	"regexp"
)

var addressPattern = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{97}$`)

type BridgeClient interface {
	bridgeTypes.Client
	EmitAssetUnsigned(data db.DepositData) (*UnsignedTransaction, error)
	EmitAssetSigned(transaction SignedTransaction) (txHash string, err error)
}

type client struct {
	logger *logan.Entry
	chain  chain.Zano
}

func (p *client) ConstructWithdrawalTx(data db.Deposit) ([]byte, error) {
	//TODO implement me
	return []byte("zano"), nil
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

func NewBridgeClient(chain chain.Zano, logger *logan.Entry) BridgeClient {
	return &client{logger, chain}
}
