package helper

import (
	"crypto/elliptic"
	"fmt"

	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/gcash/bchd/bchec"
	bchcfg "github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	bchscript "github.com/gcash/bchd/txscript"
	bchwire "github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
	"github.com/pkg/errors"
)

type bchHelper struct {
	chainParams      *bchcfg.Params
	supportedScripts map[bchscript.ScriptClass]bool
	mockKey          *bchec.PrivateKey
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

func (b *bchHelper) IsOpReturnScript(script []byte) bool {
	if len(script) == 0 {
		return false
	}
	return bchscript.GetScriptClass(script) == bchscript.NullDataTy
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

	sigHash, _, err := bchscript.CalcSignatureHash(
		scriptRaw, sigHashes, bchscript.SigHashAll,
		bchWire, idx, amt, true,
	)
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

	sigScript, err := bchscript.SignatureScript(
		bchWire, idx, amt, scriptRaw,
		bchscript.SigHashAll, b.mockKey, true,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signature script")
	}

	return sigScript, nil
}

func wireToBch(tx *btcwire.MsgTx) *bchwire.MsgTx {
	txc := &bchwire.MsgTx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
	}
	for _, rtx := range tx.TxIn {
		txi := &bchwire.TxIn{
			PreviousOutPoint: bchwire.OutPoint{
				Hash:  chainhash.Hash(rtx.PreviousOutPoint.Hash),
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
