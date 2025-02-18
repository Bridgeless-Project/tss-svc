package bitcoin

import (
	"math/big"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

const defaultFeeRate = 0.00001

func (c *Client) CreateWithdrawalTx(deposit db.Deposit, changeAddr string) (*wire.MsgTx, error) {
	amount, set := new(big.Int).SetString(*deposit.WithdrawalAmount, 10)
	if !set {
		return nil, errors.New("failed to parse amount")
	}
	receiverAddr, err := btcutil.DecodeAddress(*deposit.Receiver, c.chain.Params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode receiver address")
	}
	script, err := txscript.PayToAddrScript(receiverAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create script")
	}

	txToFund := wire.NewMsgTx(wire.TxVersion)
	txToFund.AddTxOut(wire.NewTxOut(amount.Int64(), script))

	fundOpts := btcjson.FundRawTransactionOpts{
		IncludeWatching: btcjson.Bool(true),
		ChangeAddress:   btcjson.String(changeAddr),
		ChangePosition:  btcjson.Int(0),
		FeeRate:         btcjson.Float64(defaultFeeRate),
		LockUnspents:    btcjson.Bool(false),
	}

	result, err := c.chain.Rpc.Wallet.FundRawTransaction(txToFund, fundOpts, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fund raw transaction")
	}

	// FIXME: add signHashes

	return result.Transaction, nil
}

func (c *Client) ListUnspent() ([]btcjson.ListUnspentResult, error) {
	return c.chain.Rpc.Wallet.ListUnspent()
}
