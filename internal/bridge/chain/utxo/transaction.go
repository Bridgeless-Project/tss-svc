package utxo

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/hyle-team/tss-svc/internal/bridge"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
)

type ConsolidateOutputsOptions func(*ConsolidateOutputsParams)

func WithFeeRate(rate uint64) ConsolidateOutputsOptions {
	return func(params *ConsolidateOutputsParams) {
		params.FeeRate = rate
	}
}

func WithMaxInputsCount(count int) ConsolidateOutputsOptions {
	return func(params *ConsolidateOutputsParams) {
		params.MaxInputsCount = count
	}
}

func WithOutputsCount(count int) ConsolidateOutputsOptions {
	return func(params *ConsolidateOutputsParams) {
		params.OutputsCount = count
	}
}

type ConsolidateOutputsParams struct {
	// satsPerByte
	FeeRate        uint64
	MaxInputsCount int
	OutputsCount   int
}

var DefaultConsolidateOutputsParams = ConsolidateOutputsParams{
	FeeRate:        DefaultFeeRateBtcPerKvb * 1e5,
	MaxInputsCount: 20,
	OutputsCount:   2,
}

func (c *client) GetTransaction(txHash string) (*btcjson.TxRawResult, error) {
	txHash = strings.TrimPrefix(txHash, bridge.HexPrefix)

	tx, err := c.chain.Rpc.Node.GetRawTransactionVerbose(txHash)
	if err != nil {
		if strings.Contains(err.Error(), "No such mempool or blockchain transaction") {
			return nil, bridgeTypes.ErrTxNotFound
		}

		return nil, errors.Wrap(err, "failed to get raw transaction")
	}

	return tx, nil
}

func (c *client) FindUsedInputs(tx *wire.MsgTx) (map[OutPoint]btcjson.ListUnspentResult, error) {
	unspent, err := c.ListUnspent()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available UTXOs")
	}

	usedInputs := make(map[OutPoint]btcjson.ListUnspentResult, len(tx.TxIn))
	for _, inp := range tx.TxIn {
		if inp == nil {
			return nil, errors.New("nil input in transaction")
		}

		for _, u := range unspent {
			if u.TxID != inp.PreviousOutPoint.Hash.String() ||
				u.Vout != inp.PreviousOutPoint.Index {
				continue
			}

			outPoint := OutPoint{TxID: u.TxID, Index: u.Vout}
			if _, found := usedInputs[outPoint]; found {
				return nil, errors.New(fmt.Sprintf("double spending detected for %s:%d", u.TxID, u.Vout))
			}

			usedInputs[outPoint] = u
			break
		}
	}

	if len(usedInputs) != len(tx.TxIn) {
		return nil, errors.New("not all inputs were found")
	}

	return usedInputs, nil
}

func (c *client) MockTransaction(tx *wire.MsgTx, inputs map[OutPoint]btcjson.ListUnspentResult) (*wire.MsgTx, error) {
	mockedTx := tx.Copy()

	for i, inp := range mockedTx.TxIn {
		unspent := inputs[OutPoint{TxID: inp.PreviousOutPoint.Hash.String(), Index: inp.PreviousOutPoint.Index}]
		scriptDecoded, err := hex.DecodeString(unspent.ScriptPubKey)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", i))
		}

		sig, err := c.helper.MockSignatureScript(scriptDecoded, tx, i, ToAmount(unspent.Amount).Int64())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to mock input %d", i))
		}

		mockedTx.TxIn[i].SignatureScript = sig
	}

	return mockedTx, nil
}

func (c *client) LockOutputs(tx wire.MsgTx) error {
	outs := make([]*wire.OutPoint, len(tx.TxIn))
	for i, inp := range tx.TxIn {
		outs[i] = &inp.PreviousOutPoint
	}

	return c.chain.Rpc.Wallet.LockUnspent(false, outs)
}

func (c *client) ConsolidateOutputs(to string, opts ...ConsolidateOutputsOptions) (*wire.MsgTx, [][]byte, error) {
	options := DefaultConsolidateOutputsParams
	for _, opt := range opts {
		opt(&options)
	}

	receiverScript, err := c.helper.PayToAddrScript(to)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create receiver script")
	}

	unspent, err := c.ListUnspent()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get available UTXOs")
	}
	// allow consolidation even if there is only one UTXO
	if len(unspent) == 0 {
		return nil, nil, errors.New("not enough UTXOs to consolidate")
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	for range options.OutputsCount {
		// zero value before the commission is calculated
		tx.AddTxOut(wire.NewTxOut(0, receiverScript))
	}

	limit := options.MaxInputsCount
	if limit > len(unspent) {
		limit = len(unspent)
	}

	usedInputs := make(map[OutPoint]btcjson.ListUnspentResult, limit)
	totalAmount := int64(0)
	for i := 0; i < limit; i++ {
		hash, err := chainhash.NewHashFromStr(unspent[i].TxID)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to parse tx hash for input %d", i))
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(hash, unspent[i].Vout), nil, nil))
		usedInputs[OutPoint{TxID: unspent[i].TxID, Index: unspent[i].Vout}] = unspent[i]
		totalAmount += ToAmount(unspent[i].Amount).Int64()
	}

	mockedTx, err := c.MockTransaction(tx, usedInputs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to mock transaction")
	}

	fees := int64(mockedTx.SerializeSize()) * int64(options.FeeRate)
	outAmount := totalAmount - fees

	// dividing amount equally between outputs
	amountPerOutput := outAmount / int64(options.OutputsCount)
	if !c.WithdrawalAmountValid(big.NewInt(amountPerOutput)) {
		return nil, nil, errors.New("amount per output is too small")
	}
	for _, out := range tx.TxOut {
		out.Value = amountPerOutput
	}
	// adding the remainder to the first output
	tx.TxOut[0].Value += outAmount % int64(options.OutputsCount)

	sigHashes := make([][]byte, len(tx.TxIn))
	for i := range tx.TxIn {
		utxo := usedInputs[OutPoint{
			TxID:  tx.TxIn[i].PreviousOutPoint.Hash.String(),
			Index: tx.TxIn[i].PreviousOutPoint.Index,
		}]
		scriptDecoded, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", i))
		}
		sigHash, err := c.helper.CalculateSignatureHash(scriptDecoded, tx, i, ToAmount(utxo.Amount).Int64())
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to calculate signature hash for input %d", i))
		}

		sigHashes[i] = sigHash
	}

	return tx, sigHashes, nil
}
