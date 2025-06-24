package utxo

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

const (
	// minimum fee rate is 0.00001 BTC per kilobyte
	DefaultFeeRateBtcPerKvb = 0.00002
)

type OutPoint struct {
	TxID  string
	Index uint32
}

func (c *client) CreateUnsignedWithdrawalTx(deposit db.Deposit, changeAddr string) (*wire.MsgTx, [][]byte, error) {
	amount, set := new(big.Int).SetString(deposit.WithdrawalAmount, 10)
	if !set {
		return nil, nil, errors.New("failed to parse amount")
	}
	receiverScript, err := c.helper.PayToAddrScript(deposit.Receiver)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create script")
	}

	txToFund := wire.NewMsgTx(wire.TxVersion)
	txToFund.AddTxOut(wire.NewTxOut(amount.Int64(), receiverScript))

	fundOpts := btcjson.FundRawTransactionOpts{
		IncludeWatching: btcjson.Bool(true),
		ChangeAddress:   btcjson.String(changeAddr),
		ChangePosition:  btcjson.Int(0),
		FeeRate:         btcjson.Float64(DefaultFeeRateBtcPerKvb),
		LockUnspents:    btcjson.Bool(false),
	}

	result, err := c.chain.Rpc.Wallet.FundRawTransaction(txToFund, fundOpts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fund raw transaction")
	}

	unspent, err := c.ListUnspent()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get available UTXOs")
	}

	sigHashes := make([][]byte, 0)
	for idx, inp := range result.Transaction.TxIn {
		for _, u := range unspent {
			if u.TxID == inp.PreviousOutPoint.Hash.String() {
				scriptDecoded, err := hex.DecodeString(u.ScriptPubKey)
				if err != nil {
					return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", idx))
				}
				sigHash, err := c.helper.CalculateSignatureHash(scriptDecoded, result.Transaction, idx, ToUnits(u.Amount))
				if err != nil {
					return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to calculate signature hash for input %d", idx))
				}

				sigHashes = append(sigHashes, sigHash)
				break
			}
		}
	}
	if len(sigHashes) != len(result.Transaction.TxIn) {
		return nil, nil, errors.New("failed to form enough signature hashes")
	}

	return result.Transaction, sigHashes, nil

}

func (c *client) UnspentCount() (int, error) {
	unspent, err := c.ListUnspent()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get available UTXOs")
	}

	return len(unspent), nil
}

func (c *client) ListUnspent() ([]btcjson.ListUnspentResult, error) {
	return c.chain.Rpc.Wallet.ListUnspent()
}

func (c *client) SendSignedTransaction(tx *wire.MsgTx) (string, error) {
	return c.chain.Rpc.Node.SendRawTransaction(tx)
}
