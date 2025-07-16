package client

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

func (c *client) CreateUnsignedWithdrawalTx(deposit db.Deposit, changeAddr string) (*wire.MsgTx, [][]byte, error) {
	amount, set := new(big.Int).SetString(deposit.WithdrawalAmount, 10)
	if !set {
		return nil, nil, errors.New("failed to parse amount")
	}
	receiverScript, err := c.helper.PayToAddrScript(deposit.Receiver)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create script")
	}
	receiverOutput := wire.NewTxOut(amount.Int64(), receiverScript)

	unspent, err := c.ListUnspent()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get available UTXOs")
	}

	unsignedTxData, err := c.helper.NewUnsignedTransaction(
		unspent,
		utils.DefaultFeeRateBtcPerKvb,
		[]*wire.TxOut{receiverOutput},
		changeAddr,
	)
	if err != nil {
		// TODO: check not enough funds err
		return nil, nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

	sigHashes := make([][]byte, 0)
	for i := range unsignedTxData.PrevScripts {
		sigHash, err := c.helper.CalculateSignatureHash(
			unsignedTxData.PrevScripts[i],
			unsignedTxData.Tx,
			i,
			int64(unsignedTxData.PrevInputValues[i]),
		)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to calculate signature hash for tx %d", i)
		}

		sigHashes = append(sigHashes, sigHash)
	}

	return unsignedTxData.Tx, sigHashes, nil
}

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
