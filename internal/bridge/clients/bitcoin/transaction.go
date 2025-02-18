package bitcoin

import (
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/hyle-team/tss-svc/internal/bridge"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/pkg/errors"
)

func (c *Client) GetTransaction(txHash string) (*btcjson.TxRawResult, error) {
	txHash = strings.TrimPrefix(txHash, bridge.HexPrefix)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tx hash")
	}

	tx, err := c.chain.Rpc.Node.GetRawTransactionVerbose(hash)
	if err != nil {
		if strings.Contains(err.Error(), "No such mempool or blockchain transaction") {
			return nil, bridgeTypes.ErrTxNotFound
		}
		return nil, errors.Wrap(err, "failed to get raw transaction")
	}

	return tx, nil
}
