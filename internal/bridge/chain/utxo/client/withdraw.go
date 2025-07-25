package client

import (
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

func (c *client) UnspentCount() (int, error) {
	unspent, err := c.ListUnspent()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get available UTXOs")
	}

	return len(unspent), nil
}

func (c *client) ListUnspent() ([]btcjson.ListUnspentResult, error) {
	return c.chain.Rpc.Wallet.ListUnspent(c.chain.Confirmations)
}

func (c *client) SendSignedTransaction(tx *wire.MsgTx) (string, error) {
	return c.chain.Rpc.Node.SendRawTransaction(tx)
}
