package client

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	utxochain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/factory"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
)

type Client interface {
	chain.Client

	ConsolidationThreshold() int
	FindUsedInputs(tx *wire.MsgTx) (map[types.OutPoint]btcjson.ListUnspentResult, error)
	MockTransaction(tx *wire.MsgTx, inputs map[types.OutPoint]btcjson.ListUnspentResult) (*wire.MsgTx, error)
	ConsolidateOutputs(to string, opts ...ConsolidateOutputsOptions) (*wire.MsgTx, [][]byte, error)
	UnspentCount() (int, error)
	LockOutputs(tx *wire.MsgTx) error
	ListUnspent() ([]btcjson.ListUnspentResult, error)
	SendSignedTransaction(tx *wire.MsgTx) (string, error)
	EstimateFeeOrDefault() btcutil.Amount

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
	chainHelper := factory.NewUtxoHelper(chain.Meta.Chain, chain.Meta.Network)
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
