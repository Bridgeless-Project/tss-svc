package client

import (
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

const errTxNotFound = "No such mempool or blockchain transaction"

func (c *client) GetTransaction(txHash string) (*btcjson.TxRawResult, error) {
	txHash = strings.TrimPrefix(txHash, bridge.HexPrefix)

	tx, err := c.chain.Rpc.Node.GetRawTransactionVerbose(txHash)
	if err != nil {
		if strings.Contains(err.Error(), errTxNotFound) {
			return nil, bridgeTypes.ErrTxNotFound
		}

		return nil, errors.Wrap(err, "failed to get raw transaction")
	}

	return tx, nil
}

func (c *client) LockOutputs(tx *wire.MsgTx) error {
	outs := make([]*wire.OutPoint, len(tx.TxIn))
	for i, inp := range tx.TxIn {
		outs[i] = &inp.PreviousOutPoint
	}

	return c.chain.Rpc.Wallet.LockUnspent(false, outs)
}

func (c *client) EstimateFeeOrDefault() btcutil.Amount {
	fee, err := c.chain.Rpc.Node.EstimateFee()
	switch {
	case err != nil:
		// TODO: warn about the error
		return utils.DefaultFeeRateBtcPerKvb
	case fee < utils.DefaultFeeRateBtcPerKvb:
		return utils.DefaultFeeRateBtcPerKvb
	case fee > utils.MaxFeeRateBtcPerKvb:
		return utils.MaxFeeRateBtcPerKvb
	default:
		return fee
	}
}
