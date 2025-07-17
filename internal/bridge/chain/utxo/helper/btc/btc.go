package btc

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	utxohelper "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	btccfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	btcscript "github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

type helper struct {
	chainParams      *btccfg.Params
	supportedScripts map[btcscript.ScriptClass]bool
	mockKey          *btcec.PrivateKey

	outputArranger utils.OutputArranger
}

func NewHelper(chainParams *btccfg.Params) utxohelper.UtxoHelper {
	mockedKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to create mocked private key: %v", err))
	}

	return &helper{
		chainParams: chainParams,
		mockKey:     mockedKey,
		supportedScripts: map[btcscript.ScriptClass]bool{
			btcscript.PubKeyHashTy: true,
		},
		outputArranger: utils.LargestFirstOutputArranger{},
	}
}

func (b *helper) AddressValid(addr string) bool {
	_, err := btcutil.DecodeAddress(addr, b.chainParams)
	return err == nil
}

func (b *helper) ScriptSupported(script []byte) bool {
	if len(script) == 0 {
		return false
	}

	class := btcscript.GetScriptClass(script)
	return b.supportedScripts[class]
}

func (b *helper) ExtractScriptAddresses(script []byte) ([]string, error) {
	if len(script) == 0 {
		return nil, nil
	}

	_, addresses, _, err := btcscript.ExtractPkScriptAddrs(script, b.chainParams)
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

func (b *helper) PayToAddrScript(addr string) ([]byte, error) {
	address, err := btcutil.DecodeAddress(addr, b.chainParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode address")
	}

	script, err := btcscript.PayToAddrScript(address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pay-to-address script")
	}

	return script, nil
}

func (b *helper) CalculateSignatureHash(scriptRaw []byte, tx *wire.MsgTx, idx int, _ int64) ([]byte, error) {
	if len(scriptRaw) == 0 {
		return nil, errors.New("script cannot be empty")
	}

	sigHash, err := btcscript.CalcSignatureHash(scriptRaw, btcscript.SigHashAll, tx, idx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate signature hash")
	}

	return sigHash, nil
}

func (b *helper) MockSignatureScript(scriptRaw []byte, tx *wire.MsgTx, idx int, _ int64) ([]byte, error) {
	if len(scriptRaw) == 0 {
		return nil, errors.New("script cannot be empty")
	}

	sigScript, err := btcscript.SignatureScript(tx, idx, scriptRaw, btcscript.SigHashAll, b.mockKey, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signature script")
	}

	return sigScript, nil
}

func (b *helper) InjectSignatures(tx *wire.MsgTx, signatures []*common.SignatureData, pk *ecdsa.PublicKey) error {
	if len(signatures) != len(tx.TxIn) {
		return errors.New("signatures count does not match inputs count")
	}

	for i, sig := range signatures {
		encodedSig := utils.EncodeSignature(sig, byte(btcscript.SigHashAll))
		sigScript, err := btcscript.
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

func (b *helper) P2pkhAddress(pub *ecdsa.PublicKey) string {
	compressed := crypto.CompressPubkey(pub)
	pubKeyHash := btcutil.Hash160(compressed)

	addr, _ := btcutil.NewAddressPubKeyHash(pubKeyHash, b.chainParams)

	return addr.String()
}

func (b *helper) TxHash(tx *wire.MsgTx) string {
	if tx == nil {
		return ""
	}

	return tx.TxHash().String()
}

func (b *helper) RetrieveOpReturnData(script []byte) (string, error) {
	if !btcscript.IsNullData(script) {
		return "", errors.New("invalid script type, expected valid OP_RETURN")
	}

	data, err := btcscript.PushedData(script)
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve pushed data from script")
	}
	if len(data) != 1 {
		return "", errors.New("expected exactly one pushed data item in OP_RETURN script")
	}

	return string(data[0]), nil
}

func (b *helper) NewUnsignedTransaction(
	unspent []btcjson.ListUnspentResult,
	feeRate btcutil.Amount,
	outputs []*wire.TxOut,
	changeAddr string,
) (*txauthor.AuthoredTx, error) {
	changeSource, err := b.changeSource(changeAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create change source")
	}
	arrangedOutputs := b.outputArranger.ArrangeOutputs(unspent)

	tx, err := txauthor.NewUnsignedTransaction(
		outputs,
		feeRate,
		inputSource(arrangedOutputs),
		changeSource,
	)
	if err != nil {
		// TODO: handle not enough funds error
		return nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

	return tx, nil
}

func (b *helper) changeSource(addr string) (*txauthor.ChangeSource, error) {
	changeScript, err := b.PayToAddrScript(addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create change address script")
	}

	return &txauthor.ChangeSource{
		NewScript:  func() ([]byte, error) { return changeScript, nil },
		ScriptSize: len(changeScript),
	}, nil
}

func inputSource(outputs []btcjson.ListUnspentResult) txauthor.InputSource {
	// Current inputs and their total value.
	// These are closed over by the returned input source and reused across multiple calls.
	currentTotal := btcutil.Amount(0)
	currentInputs := make([]*wire.TxIn, 0, len(outputs))
	currentScripts := make([][]byte, 0, len(outputs))
	currentInputValues := make([]btcutil.Amount, 0, len(outputs))

	return func(target btcutil.Amount) (btcutil.Amount, []*wire.TxIn, []btcutil.Amount, [][]byte, error) {
		for currentTotal < target && len(outputs) != 0 {
			out := outputs[0]

			txHash, err := chainhash.NewHashFromStr(out.TxID)
			if err != nil {
				return 0, nil, nil, nil, errors.Wrapf(err, "failed to parse tx hash %s", out.TxID)
			}
			pkScript, err := hex.DecodeString(out.ScriptPubKey)
			if err != nil {
				return 0, nil, nil, nil, errors.Wrap(err, "failed to decode script pub key")
			}

			outpoint := &wire.OutPoint{Hash: *txHash, Index: out.Vout}
			amount, _ := btcutil.NewAmount(out.Amount)

			currentInputs = append(currentInputs, wire.NewTxIn(outpoint, nil, nil))
			currentScripts = append(currentScripts, pkScript)
			currentTotal += amount
			currentInputValues = append(currentInputValues, amount)

			outputs = outputs[1:]
		}

		return currentTotal, currentInputs, currentInputValues, currentScripts, nil
	}
}
