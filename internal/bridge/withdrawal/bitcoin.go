package withdrawal

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	bitcoin2 "github.com/hyle-team/tss-svc/internal/bridge/chain/bitcoin"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	_ DepositSigningData                 = BitcoinWithdrawalData{}
	_ Constructor[BitcoinWithdrawalData] = &BitcoinWithdrawalConstructor{}
)

type BitcoinWithdrawalData struct {
	ProposalData *p2p.BitcoinProposalData
	SignedInputs [][]byte
}

func (e BitcoinWithdrawalData) DepositIdentifier() db.DepositIdentifier {
	identifier := db.DepositIdentifier{}

	if e.ProposalData == nil || e.ProposalData.DepositId == nil {
		return identifier
	}

	identifier.ChainId = e.ProposalData.DepositId.ChainId
	identifier.TxHash = e.ProposalData.DepositId.TxHash
	identifier.TxNonce = int(e.ProposalData.DepositId.TxNonce)

	return identifier
}

func (e BitcoinWithdrawalData) HashString() string {
	if e.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(e.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type BitcoinWithdrawalConstructor struct {
	client *bitcoin2.Client
	tssPkh *btcutil.AddressPubKeyHash
}

func NewBitcoinConstructor(client *bitcoin2.Client, tssPub *ecdsa.PublicKey) *BitcoinWithdrawalConstructor {
	tssPkh, err := bitcoin2.PubKeyToPkhCompressed(tssPub, client.ChainParams())
	if err != nil {
		panic(fmt.Sprintf("failed to create TSS public key hash: %v", err))
	}

	return &BitcoinWithdrawalConstructor{client: client, tssPkh: tssPkh}
}

func (c *BitcoinWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*BitcoinWithdrawalData, error) {
	tx, sigHashes, err := c.client.CreateUnsignedWithdrawalTx(deposit, c.tssPkh.EncodeAddress())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

	var buf bytes.Buffer
	if err = tx.Serialize(&buf); err != nil {
		return nil, errors.Wrap(err, "failed to serialize transaction")
	}

	return &BitcoinWithdrawalData{
		ProposalData: &p2p.BitcoinProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxNonce: uint32(deposit.TxNonce),
				TxHash:  deposit.TxHash,
			},
			SerializedTx: buf.Bytes(),
			SigData:      sigHashes,
		},
	}, nil
}

func (c *BitcoinWithdrawalConstructor) IsValid(data BitcoinWithdrawalData, deposit db.Deposit) (bool, error) {
	tx := wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(data.ProposalData.SerializedTx)); err != nil {
		return false, errors.Wrap(err, "failed to deserialize transaction")
	}

	outputsSum, err := c.validateOutputs(&tx, deposit)
	if err != nil {
		return false, errors.Wrap(err, "failed to validate outputs")
	}

	usedInputs, err := c.client.FindUsedInputs(&tx)
	if err != nil {
		return false, errors.Wrap(err, "failed to find used inputs")
	}

	inputsSum, err := c.validateInputs(&tx, usedInputs, data.ProposalData.SigData)
	if err != nil {
		return false, errors.Wrap(err, "failed to validate inputs")
	}

	if err = c.validateChange(&tx, usedInputs, inputsSum, outputsSum); err != nil {
		return false, errors.Wrap(err, "failed to validate change")
	}

	return true, nil
}

func (c *BitcoinWithdrawalConstructor) validateOutputs(tx *wire.MsgTx, deposit db.Deposit) (int64, error) {
	outputsSum, receiverIdx := int64(0), 0
	switch len(tx.TxOut) {
	case 2:
		// 1st output is for the change, 2nd is for the receiver
		receiverIdx = 1
		changeOutput := tx.TxOut[0]
		outputsSum += changeOutput.Value

		outScript, err := txscript.PayToAddrScript(c.tssPkh)
		if err != nil {
			return 0, errors.Wrap(err, "failed to create change output script")
		}
		if !bytes.Equal(changeOutput.PkScript, outScript) {
			return 0, errors.New("invalid change output script")
		}

		fallthrough
	case 1:
		receiverOutput := tx.TxOut[receiverIdx]
		withdrawalAmount, ok := new(big.Int).SetString(deposit.WithdrawalAmount, 10)
		if !ok || receiverOutput.Value != withdrawalAmount.Int64() {
			return 0, errors.New("invalid withdrawal amount")
		}
		outputsSum += receiverOutput.Value

		outAddr, err := btcutil.DecodeAddress(deposit.Receiver, c.client.ChainParams())
		if err != nil {
			return 0, errors.Wrap(err, "failed to decode receiver address")
		}
		outScript, err := txscript.PayToAddrScript(outAddr)
		if err != nil {
			return 0, errors.Wrap(err, "failed to create change output script")
		}
		if !bytes.Equal(receiverOutput.PkScript, outScript) {
			return 0, errors.New("invalid receiver output script")
		}
	default:
		return 0, errors.New("invalid number of transaction outputs")
	}

	return outputsSum, nil
}

func (c *BitcoinWithdrawalConstructor) validateInputs(
	tx *wire.MsgTx,
	inputs map[bitcoin2.OutPoint]btcjson.ListUnspentResult,
	sigHashes [][]byte,
) (int64, error) {
	if sigHashes == nil || len(sigHashes) != len(tx.TxIn) {
		return 0, errors.New("invalid signature hashes")
	}

	inputsSum := int64(0)
	for idx, inp := range tx.TxIn {
		if inp == nil {
			return 0, errors.New(fmt.Sprintf("nil input at index %d", idx))
		}

		unspent := inputs[bitcoin2.OutPoint{TxID: inp.PreviousOutPoint.Hash.String(), Index: inp.PreviousOutPoint.Index}]

		scriptDecoded, err := hex.DecodeString(unspent.ScriptPubKey)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", idx))
		}
		sigHash, err := txscript.CalcSignatureHash(scriptDecoded, bitcoin2.SigHashType, tx, idx)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("failed to calculate signature hash for input %d", idx))
		}
		if !bytes.Equal(sigHashes[idx], sigHash) {
			return 0, errors.New(fmt.Sprintf("invalid signature hash for input %d", idx))
		}

		inputsSum += bitcoin2.ToAmount(unspent.Amount, bitcoin2.Decimals).Int64()
	}

	return inputsSum, nil
}

func (c *BitcoinWithdrawalConstructor) validateChange(tx *wire.MsgTx, inputs map[bitcoin2.OutPoint]btcjson.ListUnspentResult, inputsSum, outputsSum int64) error {
	actualFee := inputsSum - outputsSum
	if actualFee <= 0 {
		return errors.New("invalid change amount")
	}

	mockedTx, err := c.client.MockTransaction(tx, inputs)
	if err != nil {
		return errors.Wrap(err, "failed to mock transaction")
	}

	var (
		targetFeeRate = bitcoin2.DefaultFeeRateBtcPerKvb * 1e5 // btc/kB -> sat/byte
		feeTolerance  = 0.1 * targetFeeRate                    // 10%
		estimatedSize = mockedTx.SerializeSize()
		actualFeeRate = float64(actualFee) / float64(estimatedSize)
	)

	if math.Abs(actualFeeRate-targetFeeRate) > feeTolerance {
		return errors.New(fmt.Sprintf("provided fee rate %f is not within %f of target %f", actualFeeRate, feeTolerance, targetFeeRate))
	}

	return nil
}
