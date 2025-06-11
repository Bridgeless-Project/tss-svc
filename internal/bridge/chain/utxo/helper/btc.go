package helper

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	btccfg "github.com/btcsuite/btcd/chaincfg"
	btcscript "github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

type btcHelper struct {
	chainParams      *btccfg.Params
	supportedScripts map[btcscript.ScriptClass]bool
	mockKey          *btcec.PrivateKey
}

func NewBtcHelper(chainParams *btccfg.Params) UtxoHelper {
	mockedKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to create mocked private key: %v", err))
	}

	return &btcHelper{
		chainParams: chainParams,
		mockKey:     mockedKey,
		// TODO: review supported scripts
		supportedScripts: map[btcscript.ScriptClass]bool{
			btcscript.PubKeyHashTy: true,
		},
	}
}

func (b *btcHelper) AddressValid(addr string) bool {
	_, err := btcutil.DecodeAddress(addr, b.chainParams)
	return err == nil
}

func (b *btcHelper) ScriptSupported(script []byte) bool {
	if len(script) == 0 {
		return false
	}

	class := btcscript.GetScriptClass(script)
	return b.supportedScripts[class]
}

func (b *btcHelper) ExtractScriptAddresses(script []byte) ([]string, error) {
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

func (b *btcHelper) IsOpReturnScript(script []byte) bool {
	if len(script) == 0 {
		return false
	}
	return btcscript.GetScriptClass(script) == btcscript.NullDataTy
}

func (b *btcHelper) PayToAddrScript(addr string) ([]byte, error) {
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

func (b *btcHelper) CalculateSignatureHash(scriptRaw []byte, tx *wire.MsgTx, idx int, _ int64) ([]byte, error) {
	if len(scriptRaw) == 0 {
		return nil, errors.New("script cannot be empty")
	}

	sigHash, err := btcscript.CalcSignatureHash(scriptRaw, btcscript.SigHashAll, tx, idx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate signature hash")
	}

	return sigHash, nil
}

func (b *btcHelper) MockSignatureScript(scriptRaw []byte, tx *wire.MsgTx, idx int, amt int64) ([]byte, error) {
	if len(scriptRaw) == 0 {
		return nil, errors.New("script cannot be empty")
	}

	sigScript, err := btcscript.SignatureScript(tx, idx, scriptRaw, btcscript.SigHashAll, b.mockKey, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signature script")
	}

	return sigScript, nil
}
