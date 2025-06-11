package utxo

import (
	"math/big"

	"github.com/btcsuite/btcd/wire"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	utxochain "github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/hyle-team/tss-svc/internal/db"
)

type Client interface {
	chain.Client

	ConsolidationThreshold() int
	CreateUnsignedWithdrawalTx(deposit db.Deposit, changeAddr string) (*wire.MsgTx, [][]byte, error)

	UtxoHelper() helper.UtxoHelper
}

const ConsolidationThreshold = 20

var dustAmount = big.NewInt(547)

type client struct {
	chain          utxochain.Chain
	depositDecoder *DepositDecoder
	helper         helper.UtxoHelper
}

func NewBridgeClient(chain utxochain.Chain) Client {
	chainHelper := helper.NewUtxoHelper(chain.Meta.Type, chain.Meta.Network)
	return &client{
		chain:          chain,
		helper:         chainHelper,
		depositDecoder: NewDepositDecoder(chainHelper, chain.Receivers),
	}
}

func (c *client) UtxoHelper() helper.UtxoHelper {
	return c.helper
}

func (c *client) ChainId() string {
	return c.chain.Id
}

func (c *client) Type() chain.Type {
	return chain.TypeBitcoin
}

func (c *client) ConsolidationThreshold() int {
	return ConsolidationThreshold
}

func (c *client) AddressValid(addr string) bool {
	return c.helper.AddressValid(addr)
}

func (c *client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func (c *client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(dustAmount) == -1 {
		return false
	}

	return true
}
