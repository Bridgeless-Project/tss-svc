package withdrawal

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	_ DepositSigningData              = UtxoWithdrawalData{}
	_ Constructor[UtxoWithdrawalData] = &UtxoWithdrawalConstructor{}
)

type UtxoWithdrawalData struct {
	ProposalData *p2p.BitcoinProposalData
}

func (e UtxoWithdrawalData) DepositIdentifier() db.DepositIdentifier {
	identifier := db.DepositIdentifier{}

	if e.ProposalData == nil || e.ProposalData.DepositId == nil {
		return identifier
	}

	identifier.ChainId = e.ProposalData.DepositId.ChainId
	identifier.TxHash = e.ProposalData.DepositId.TxHash
	identifier.TxNonce = int(e.ProposalData.DepositId.TxNonce)

	return identifier
}

func (e UtxoWithdrawalData) HashString() string {
	if e.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(e.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type UtxoWithdrawalConstructor struct {
	client  utxo.Client
	helper  helper.UtxoHelper
	tssAddr string
}

func NewBitcoinConstructor(client utxo.Client, tssPub *ecdsa.PublicKey) *UtxoWithdrawalConstructor {
	hlp := client.UtxoHelper()
	return &UtxoWithdrawalConstructor{
		client:  client,
		helper:  hlp,
		tssAddr: hlp.P2pkhAddress(tssPub),
	}
}

func (c *UtxoWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*UtxoWithdrawalData, error) {
	tx, sigHashes, err := c.client.CreateUnsignedWithdrawalTx(deposit, c.tssAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

	var buf bytes.Buffer
	if err = tx.Serialize(&buf); err != nil {
		return nil, errors.Wrap(err, "failed to serialize transaction")
	}

	return &UtxoWithdrawalData{
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

func (c *UtxoWithdrawalConstructor) IsValid(data UtxoWithdrawalData, deposit db.Deposit) (bool, error) {
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

func (c *UtxoWithdrawalConstructor) validateOutputs(tx *wire.MsgTx, deposit db.Deposit) (int64, error) {
	outputsSum, receiverIdx := int64(0), 0
	switch len(tx.TxOut) {
	case 2:
		// 1st output is for the change, 2nd is for the receiver
		receiverIdx = 1
		changeOutput := tx.TxOut[0]
		outputsSum += changeOutput.Value

		outScript, err := c.helper.PayToAddrScript(c.tssAddr)
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

		outScript, err := c.helper.PayToAddrScript(deposit.Receiver)
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

func (c *UtxoWithdrawalConstructor) validateInputs(
	tx *wire.MsgTx,
	inputs map[utxo.OutPoint]btcjson.ListUnspentResult,
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

		unspent := inputs[utxo.OutPoint{TxID: inp.PreviousOutPoint.Hash.String(), Index: inp.PreviousOutPoint.Index}]
		unspentAmount := utxo.ToAmount(unspent.Amount).Int64()

		scriptDecoded, err := hex.DecodeString(unspent.ScriptPubKey)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", idx))
		}
		sigHash, err := c.helper.CalculateSignatureHash(scriptDecoded, tx, idx, unspentAmount)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("failed to calculate signature hash for input %d", idx))
		}
		if !bytes.Equal(sigHashes[idx], sigHash) {
			return 0, errors.New(fmt.Sprintf("invalid signature hash for input %d", idx))
		}

		inputsSum += unspentAmount
	}

	return inputsSum, nil
}

func (c *UtxoWithdrawalConstructor) validateChange(
	tx *wire.MsgTx,
	inputs map[utxo.OutPoint]btcjson.ListUnspentResult,
	inputsSum,
	outputsSum int64,
) error {
	actualFee := inputsSum - outputsSum
	if actualFee <= 0 {
		return errors.New("invalid change amount")
	}

	mockedTx, err := c.client.MockTransaction(tx, inputs)
	if err != nil {
		return errors.Wrap(err, "failed to mock transaction")
	}

	var (
		targetFeeRate = utxo.DefaultFeeRateBtcPerKvb * 1e5 // btc/kB -> sat/byte
		feeTolerance  = 0.1 * targetFeeRate                // 10%
		estimatedSize = mockedTx.SerializeSize()
		actualFeeRate = float64(actualFee) / float64(estimatedSize)
	)

	if math.Abs(actualFeeRate-targetFeeRate) > feeTolerance {
		return errors.New(fmt.Sprintf("provided fee rate %f is not within %f of target %f", actualFeeRate, feeTolerance, targetFeeRate))
	}

	return nil
}
