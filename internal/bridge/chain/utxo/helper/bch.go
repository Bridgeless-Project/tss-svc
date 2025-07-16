package helper

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	btctxauthor "github.com/btcsuite/btcwallet/wallet/txauthor"
	bchtxauthor "github.com/gcash/bchwallet/wallet/txauthor"

	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchd/bchec"

	btcchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	bchcfg "github.com/gcash/bchd/chaincfg"
	bchchainhash "github.com/gcash/bchd/chaincfg/chainhash"
	bchscript "github.com/gcash/bchd/txscript"
	bchwire "github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
	"github.com/pkg/errors"
)

const sigHashType = bchscript.SigHashAll | bchscript.SigHashForkID

type bchHelper struct {
	chainParams      *bchcfg.Params
	supportedScripts map[bchscript.ScriptClass]bool
	mockKey          *bchec.PrivateKey

	outputArranger OutputArranger
}

func NewBchHelper(chainParams *bchcfg.Params) UtxoHelper {
	mockedKey, err := bchec.NewPrivateKey(elliptic.P256())
	if err != nil {
		panic(fmt.Sprintf("failed to create mocked private key: %v", err))
	}

	return &bchHelper{
		chainParams: chainParams,
		mockKey:     mockedKey,
		supportedScripts: map[bchscript.ScriptClass]bool{
			// TODO: review supported scripts
			bchscript.PubKeyHashTy: true,
		},
		outputArranger: LargestFirstOutputArranger{},
	}
}

func (b *bchHelper) ScriptSupported(script []byte) bool {
	if len(script) == 0 {
		return false
	}

	class := bchscript.GetScriptClass(script)
	return b.supportedScripts[class]
}

func (b *bchHelper) ExtractScriptAddresses(script []byte) ([]string, error) {
	if len(script) == 0 {
		return nil, nil
	}

	_, addresses, _, err := bchscript.ExtractPkScriptAddrs(script, b.chainParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract addresses from script pub key")
	}
	if len(addresses) == 0 {
		return nil, errors.New("no addresses found in script pub key")
	}

	addrs := make([]string, len(addresses))
	for i, addr := range addresses {
		addrs[i] = addr.String()
	}

	return addrs, nil
}

func (b *bchHelper) AddressValid(addr string) bool {
	_, err := bchutil.DecodeAddress(addr, b.chainParams)
	return err == nil
}

func (b *bchHelper) PayToAddrScript(addr string) ([]byte, error) {
	address, err := bchutil.DecodeAddress(addr, b.chainParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode address")
	}

	script, err := bchscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pay-to-address script")
	}

	return script, nil
}

func (b *bchHelper) CalculateSignatureHash(scriptRaw []byte, tx *btcwire.MsgTx, idx int, amt int64) ([]byte, error) {
	if len(scriptRaw) == 0 {
		return nil, errors.New("script cannot be empty")
	}

	bchWire := wireToBch(tx)
	sigHashes := bchscript.NewTxSigHashes(bchWire)

	sigHash, _, err := bchscript.CalcSignatureHash(scriptRaw, sigHashes, sigHashType, bchWire, idx, amt, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate signature hash")
	}

	return sigHash, nil
}

func (b *bchHelper) MockSignatureScript(scriptRaw []byte, tx *btcwire.MsgTx, idx int, amt int64) ([]byte, error) {
	if len(scriptRaw) == 0 {
		return nil, errors.New("script cannot be empty")
	}

	bchWire := wireToBch(tx)

	sigScript, err := bchscript.SignatureScript(bchWire, idx, amt, scriptRaw, sigHashType, b.mockKey, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signature script")
	}

	return sigScript, nil
}

func (b *bchHelper) P2pkhAddress(pk *ecdsa.PublicKey) string {
	compressed := crypto.CompressPubkey(pk)
	pubKeyHash := bchutil.Hash160(compressed)

	addr, _ := bchutil.NewAddressPubKeyHash(pubKeyHash, b.chainParams)

	return addr.String()
}

func (b *bchHelper) InjectSignatures(tx *btcwire.MsgTx, signatures []*common.SignatureData, pk *ecdsa.PublicKey) error {
	if len(signatures) != len(tx.TxIn) {
		return errors.New("signatures count does not match inputs count")
	}

	for i, sig := range signatures {
		encodedSig := EncodeSignature(sig, byte(sigHashType))
		sigScript, err := bchscript.
			NewScriptBuilder().
			AddData(encodedSig).
			AddData(crypto.CompressPubkey(pk)).
			Script()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to create script for input %d", i))
		}

		tx.TxIn[i].SignatureScript = sigScript
	}

	return nil
}

func (b *bchHelper) TxHash(tx *btcwire.MsgTx) string {
	if tx == nil {
		return ""
	}

	bchWire := wireToBch(tx)
	return bchWire.TxHash().String()
}

func (b *bchHelper) RetrieveOpReturnData(script []byte) (string, error) {
	if bchscript.GetScriptClass(script) != bchscript.NullDataTy {
		return "", errors.New("invalid script type, expected valid OP_RETURN")
	}

	data, err := bchscript.PushedData(script)
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve pushed data from script")
	}

	if len(data) != 1 {
		return "", errors.New("expected exactly one pushed data item in OP_RETURN script")
	}

	return string(data[0]), nil
}

func (b *bchHelper) NewUnsignedTransaction(
	unspent []btcjson.ListUnspentResult,
	feeRate btcutil.Amount,
	outputs []*btcwire.TxOut,
	changeAddr string,
) (*btctxauthor.AuthoredTx, error) {
	changeSource, err := b.changeSource(changeAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create change source")
	}

	arranged := b.outputArranger.ArrangeOutputs(unspent)
	inputSource := inputSourceBch(arranged)

	tx, err := bchtxauthor.NewUnsignedTransaction(
		outputsToBch(outputs),
		bchutil.Amount(feeRate),
		inputSource,
		changeSource,
	)
	if err != nil {
		// TODO: handle not enough funds error
		return nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

	return txAuthorToBtc(tx), nil
}

func outputsToBch(outputs []*btcwire.TxOut) []*bchwire.TxOut {
	bchOutputs := make([]*bchwire.TxOut, len(outputs))
	for i, out := range outputs {
		bchOutputs[i] = &bchwire.TxOut{
			Value:    out.Value,
			PkScript: out.PkScript,
		}
	}
	return bchOutputs
}

func txAuthorToBtc(tx *bchtxauthor.AuthoredTx) *btctxauthor.AuthoredTx {
	btcTx := &btctxauthor.AuthoredTx{
		Tx:          bchToWire(tx.Tx),
		PrevScripts: tx.PrevScripts,
		TotalInput:  btcutil.Amount(tx.TotalInput),
		ChangeIndex: tx.ChangeIndex,
	}

	prevInputValues := make([]btcutil.Amount, len(tx.PrevInputValues))
	for i, val := range tx.PrevInputValues {
		prevInputValues[i] = btcutil.Amount(val)
	}
	btcTx.PrevInputValues = prevInputValues

	return btcTx
}

func bchToWire(tx *bchwire.MsgTx) *btcwire.MsgTx {
	btcTx := &btcwire.MsgTx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
	}

	for _, rtx := range tx.TxIn {
		txi := &btcwire.TxIn{
			PreviousOutPoint: btcwire.OutPoint{
				Hash:  btcchainhash.Hash(rtx.PreviousOutPoint.Hash),
				Index: rtx.PreviousOutPoint.Index,
			},
			SignatureScript: rtx.SignatureScript,
			Sequence:        rtx.Sequence,
		}
		btcTx.TxIn = append(btcTx.TxIn, txi)
	}
	for _, stx := range tx.TxOut {
		txo := &btcwire.TxOut{
			Value:    stx.Value,
			PkScript: stx.PkScript,
		}
		btcTx.TxOut = append(btcTx.TxOut, txo)
	}

	return btcTx
}

func wireToBch(tx *btcwire.MsgTx) *bchwire.MsgTx {
	txc := &bchwire.MsgTx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
	}

	for _, rtx := range tx.TxIn {
		txi := &bchwire.TxIn{
			PreviousOutPoint: bchwire.OutPoint{
				Hash:  bchchainhash.Hash(rtx.PreviousOutPoint.Hash),
				Index: rtx.PreviousOutPoint.Index,
			},
			SignatureScript: rtx.SignatureScript,
			Sequence:        rtx.Sequence,
		}
		txc.TxIn = append(txc.TxIn, txi)
	}
	for _, stx := range tx.TxOut {
		txo := &bchwire.TxOut{
			Value:    stx.Value,
			PkScript: stx.PkScript,
		}
		txc.TxOut = append(txc.TxOut, txo)
	}

	return txc
}

func (b *bchHelper) changeSource(addr string) (bchtxauthor.ChangeSource, error) {
	changeScript, err := b.PayToAddrScript(addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create change address script")
	}

	return func() ([]byte, error) { return changeScript, nil }, nil
}

func inputSourceBch(outputs []btcjson.ListUnspentResult) bchtxauthor.InputSource {
	// Current inputs and their total value.
	// These are closed over by the returned input source and reused across multiple calls.
	currentTotal := bchutil.Amount(0)
	currentInputs := make([]*bchwire.TxIn, 0, len(outputs))
	currentScripts := make([][]byte, 0, len(outputs))
	currentInputValues := make([]bchutil.Amount, 0, len(outputs))

	return func(target bchutil.Amount) (bchutil.Amount, []*bchwire.TxIn, []bchutil.Amount, [][]byte, error) {
		for currentTotal < target && len(outputs) != 0 {
			out := outputs[0]

			txHash, err := bchchainhash.NewHashFromStr(out.TxID)
			if err != nil {
				return 0, nil, nil, nil, errors.Wrapf(err, "failed to parse tx hash %s", out.TxID)
			}
			pkScript, err := hex.DecodeString(out.ScriptPubKey)
			if err != nil {
				return 0, nil, nil, nil, errors.Wrap(err, "failed to decode script pub key")
			}

			outpoint := &bchwire.OutPoint{Hash: *txHash, Index: out.Vout}
			amount := bchutil.Amount(out.Amount)

			currentInputs = append(currentInputs, bchwire.NewTxIn(outpoint, nil))
			currentScripts = append(currentScripts, pkScript)
			currentTotal += amount
			currentInputValues = append(currentInputValues, amount)

			outputs = outputs[1:]
		}

		return currentTotal, currentInputs, currentInputValues, currentScripts, nil
	}
}
