package client

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	utxochain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/factory"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

type Client interface {
	chain.Client

	ConsolidationThreshold() int
	UnspentCount() (int, error)
	LockOutputs(tx *wire.MsgTx) error
	ListUnspent() ([]btcjson.ListUnspentResult, error)
	SendSignedTransaction(tx *wire.MsgTx) (string, error)
	EstimateFeeOrDefault() btcutil.Amount

	UtxoHelper() helper.UtxoHelper
}

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
	return utils.ConsolidationThreshold
}

func (c *client) AddressValid(addr string) bool {
	return c.helper.AddressValid(addr)
}

func (c *client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func (c *client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(utils.DustAmount) == -1 {
		return false
	}

	return true
}

func (c *client) HealthCheck() error {
	_, err := c.chain.Rpc.Node.GetBlockCount()
	if err != nil {
		return errors.Wrap(err, "failed to check node health")
	}

	_, err = c.chain.Rpc.Wallet.GetWalletInfo()
	if err != nil {
		return errors.Wrap(err, "failed to check wallet health")
	}

	return nil
}
