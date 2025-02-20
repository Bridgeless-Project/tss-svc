package bitcoin

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

const (
	DefaultFeeRateBtcPerKvb = 0.00001
	SigHashType             = txscript.SigHashAll
)

func (c *Client) CreateUnsignedWithdrawalTx(deposit db.Deposit, changeAddr string) (*wire.MsgTx, [][]byte, error) {
	amount, set := new(big.Int).SetString(*deposit.WithdrawalAmount, 10)
	if !set {
		return nil, nil, errors.New("failed to parse amount")
	}
	receiverAddr, err := btcutil.DecodeAddress(*deposit.Receiver, c.chain.Params)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode receiver address")
	}
	script, err := txscript.PayToAddrScript(receiverAddr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create script")
	}

	txToFund := wire.NewMsgTx(wire.TxVersion)
	txToFund.AddTxOut(wire.NewTxOut(amount.Int64(), script))

	fundOpts := btcjson.FundRawTransactionOpts{
		IncludeWatching: btcjson.Bool(true),
		ChangeAddress:   btcjson.String(changeAddr),
		ChangePosition:  btcjson.Int(0),
		FeeRate:         btcjson.Float64(DefaultFeeRateBtcPerKvb),
		LockUnspents:    btcjson.Bool(false),
	}

	result, err := c.chain.Rpc.Wallet.FundRawTransaction(txToFund, fundOpts, nil)
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
				sigHash, err := txscript.CalcSignatureHash(scriptDecoded, SigHashType, result.Transaction, idx)
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

func (c *Client) ListUnspent() ([]btcjson.ListUnspentResult, error) {
	return c.chain.Rpc.Wallet.ListUnspent()
}

func (c *Client) SendSignedTransaction(tx *wire.MsgTx) (*chainhash.Hash, error) {
	return c.chain.Rpc.Node.SendRawTransaction(tx, false)
}
