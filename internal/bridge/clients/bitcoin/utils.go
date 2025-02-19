package bitcoin

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/tendermint/btcd/btcec"
	"github.com/tendermint/btcd/txscript"
)

const Decimals = 8

func InjectSignatures(tx *wire.MsgTx, signatures []*common.SignatureData, pk []byte) error {
	if len(signatures) != len(tx.TxIn) {
		return errors.New("signatures count does not match inputs count")
	}

	for i, sig := range signatures {
		encodedSig := EncodeSignature(sig)
		sigScript, err := txscript.
			NewScriptBuilder().
			AddData(encodedSig).
			AddData(pk).
			Script()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to create script for input %d", i))
		}

		tx.TxIn[i].SignatureScript = sigScript
	}

	return nil
}

func EncodeSignature(sig *common.SignatureData) []byte {
	if sig == nil {
		return nil
	}

	btcSig := btcec.Signature{
		R: new(big.Int).SetBytes(sig.R),
		S: new(big.Int).SetBytes(sig.S),
	}

	return append(btcSig.Serialize(), byte(SigHashType))
}

func PubKeyToPkhCompressed(pub *ecdsa.PublicKey, chainParams *chaincfg.Params) (*btcutil.AddressPubKeyHash, error) {
	compressed := crypto.CompressPubkey(pub)
	pubKeyHash := btcutil.Hash160(compressed)

	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, chainParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create address")
	}

	return addr, nil
}

func ToAmount(val float64, decimals int64) *big.Int {
	bigval := new(big.Float).SetFloat64(val)

	coin := new(big.Float)
	coin.SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(decimals), nil))

	bigval.Mul(bigval, coin)

	result := new(big.Int)
	bigval.Int(result)

	return result
}
