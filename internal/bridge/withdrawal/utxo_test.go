package withdrawal

import (
	"crypto/ecdsa"
	"math/big"
	"os"
	"testing"

	utxochain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/chain"
	utxo "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	utxotypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	constructor    *UtxoWithdrawalConstructor
	utxoclient     utxo.Client
	receiverScript []byte
	pk             *ecdsa.PublicKey
)

func TestMain(m *testing.M) {
	utxoclient = utxo.NewBridgeClient(utxochain.Chain{
		Meta: utxochain.Meta{
			Chain:   utxotypes.ChainBtc,
			Network: utxotypes.NetworkTestnet3,
		},
	})

	x, _ := new(big.Int).SetString("22621624156344751680357860251605965662037480693022391086310495909822437991670", 0)
	y, _ := new(big.Int).SetString("86759957204076736529225308563412488286508212350611802309616991897309834072396", 0)
	pk = &ecdsa.PublicKey{
		Curve: crypto.S256(),
		X:     x,
		Y:     y,
	}

	constructor = NewUtxoConstructor(utxoclient, pk)
	receiverScript, _ = utxoclient.UtxoHelper().PayToAddrScript(utxoclient.UtxoHelper().P2pkhAddress(pk))

	code := m.Run()
	os.Exit(code)
}

func baseTx() *wire.MsgTx {
	txhash, _ := chainhash.NewHashFromStr("05837c626141191c42210876396c846dfd604dc32e219c82c517ebac8bd7d0eb")
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{PreviousOutPoint: wire.OutPoint{Hash: *txhash, Index: 0}})
	return tx
}

func Test_SubtractFee(t *testing.T) {
	type testCase struct {
		tx                 func() *txauthor.AuthoredTx
		feeRate            btcutil.Amount
		expectedWithdrawal btcutil.Amount
		changeExpected     bool
		expectedChange     btcutil.Amount
	}

	tcs := map[string]testCase{
		"must subtract ordinary fee": {
			tx: func() *txauthor.AuthoredTx {
				tx := baseTx() // 1 inp 2 outs comm 227
				tx.AddTxOut(&wire.TxOut{
					Value:    5000,
					PkScript: receiverScript,
				})
				tx.AddTxOut(&wire.TxOut{ // change output
					Value:    2773,
					PkScript: receiverScript,
				})

				return &txauthor.AuthoredTx{
					Tx:          tx,
					TotalInput:  8000,
					ChangeIndex: 1,
				}
			},
			feeRate:            1000,
			expectedWithdrawal: 4773,
			changeExpected:     true,
			expectedChange:     btcutil.Amount(3000),
		},
		"must add valid change output": {
			tx: func() *txauthor.AuthoredTx {
				tx := baseTx() // 1 inp 1 out comm 193 + 34 for new change output
				tx.AddTxOut(&wire.TxOut{
					Value:    5000,
					PkScript: receiverScript,
				})

				return &txauthor.AuthoredTx{
					Tx:          tx,
					TotalInput:  5547,
					ChangeIndex: -1,
				}
			},
			feeRate:            1000,
			expectedWithdrawal: 4773,
			changeExpected:     true,
			expectedChange:     btcutil.Amount(547),
		},
		"must not add invalid change output": {
			tx: func() *txauthor.AuthoredTx {
				tx := baseTx() // 1 inp 1 out comm 193
				tx.AddTxOut(&wire.TxOut{
					Value:    5000,
					PkScript: receiverScript,
				})

				return &txauthor.AuthoredTx{
					Tx:          tx,
					TotalInput:  5200,
					ChangeIndex: -1,
				}
			},
			feeRate:            1000,
			expectedWithdrawal: 4807,
			changeExpected:     false,
		},
		"must not charge anything if invalid withdrawal amount": {
			tx: func() *txauthor.AuthoredTx {
				tx := baseTx() // 1 inp 1 out comm 193
				tx.AddTxOut(&wire.TxOut{
					Value:    600,
					PkScript: receiverScript,
				})

				return &txauthor.AuthoredTx{
					Tx:          tx,
					TotalInput:  1000,
					ChangeIndex: -1,
				}
			},
			feeRate:            1000,
			expectedWithdrawal: 600,
			changeExpected:     false,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			finalTx := constructor.subtractFeeFromWithdrawal(tc.tx(), tc.feeRate)

			if finalTx.Tx.TxOut[0].Value != int64(tc.expectedWithdrawal) {
				t.Fatalf("expected withdrawal output value to be %d, got %d", tc.expectedWithdrawal, finalTx.Tx.TxOut[0].Value)
			}

			if tc.changeExpected {
				if finalTx.Tx.TxOut[1].Value != int64(tc.expectedChange) {
					t.Fatalf("expected change output value to be %d, got %d", tc.expectedChange, finalTx.Tx.TxOut[1].Value)
				}
			} else {
				if len(finalTx.Tx.TxOut) != 1 || finalTx.ChangeIndex != -1 {
					t.Fatalf("expected no change output, but got %d outputs", len(finalTx.Tx.TxOut))
				}
			}
		})
	}
}
