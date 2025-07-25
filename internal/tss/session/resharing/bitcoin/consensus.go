package bitcoin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/bitcoin"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	_ consensus.SigningData            = SigningData{}
	_ consensus.Mechanism[SigningData] = &ConsensusMechanism{}
)

type SigningData struct {
	ProposalData *p2p.BitcoinResharingProposalData
}

func (s SigningData) HashString() string {
	if s.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type ConsensusMechanism struct {
	client  *bitcoin.Client
	dstAddr btcutil.Address
	params  bitcoin.ConsolidateOutputsParams
}

func NewConsensusMechanism(client *bitcoin.Client, dst btcutil.Address, params bitcoin.ConsolidateOutputsParams) *ConsensusMechanism {
	return &ConsensusMechanism{client, dst, params}
}

func (m *ConsensusMechanism) FormProposalData() (*SigningData, error) {
	tx, sigHashes, err := m.client.ConsolidateOutputs(
		m.dstAddr,
		bitcoin.WithFeeRate(m.params.FeeRate),
		bitcoin.WithOutputsCount(m.params.OutputsCount),
		bitcoin.WithMaxInputsCount(m.params.MaxInputsCount),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to consolidate outputs")
	}

	var buf bytes.Buffer
	if err = tx.Serialize(&buf); err != nil {
		return nil, errors.Wrap(err, "failed to serialize transaction")
	}

	return &SigningData{
		ProposalData: &p2p.BitcoinResharingProposalData{
			SerializedTx: buf.Bytes(),
			SigData:      sigHashes,
		},
	}, nil
}

func (m *ConsensusMechanism) VerifyProposedData(data SigningData) error {
	tx := wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(data.ProposalData.SerializedTx)); err != nil {
		return errors.Wrap(err, "failed to deserialize transaction")
	}

	outputsSum, err := m.validateOutputs(&tx)
	if err != nil {
		return errors.Wrap(err, "failed to validate outputs")
	}

	usedInputs, err := m.client.FindUsedInputs(&tx)
	if err != nil {
		return errors.Wrap(err, "failed to find used inputs")
	}

	inputsSum, err := m.validateInputs(&tx, usedInputs, data.ProposalData.SigData)
	if err != nil {
		return errors.Wrap(err, "failed to validate inputs")
	}

	if err = m.validateChange(&tx, usedInputs, inputsSum, outputsSum); err != nil {
		return errors.Wrap(err, "failed to validate change")
	}

	return nil
}

func (m *ConsensusMechanism) validateOutputs(tx *wire.MsgTx) (int64, error) {
	var outputsSum int64

	targetScript, err := txscript.PayToAddrScript(m.dstAddr)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create target script")
	}

	for i, output := range tx.TxOut {
		if output == nil {
			return 0, errors.New(fmt.Sprintf("nil output at index %d", i))
		}
		if !bytes.Equal(output.PkScript, targetScript) {
			return 0, errors.New(fmt.Sprintf("unexpected output script at index %d", i))
		}

		outputsSum += output.Value
	}

	return outputsSum, nil
}

func (m *ConsensusMechanism) validateInputs(
	tx *wire.MsgTx,
	inputs map[bitcoin.OutPoint]btcjson.ListUnspentResult,
	sigHashes [][]byte,
) (int64, error) {
	var inputsSum int64

	for idx, inp := range tx.TxIn {
		if inp == nil {
			return 0, errors.New(fmt.Sprintf("nil input at index %d", idx))
		}

		unspent := inputs[bitcoin.OutPoint{TxID: inp.PreviousOutPoint.Hash.String(), Index: inp.PreviousOutPoint.Index}]

		scriptDecoded, err := hex.DecodeString(unspent.ScriptPubKey)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", idx))
		}
		sigHash, err := txscript.CalcSignatureHash(scriptDecoded, bitcoin.SigHashType, tx, idx)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("failed to calculate signature hash for input %d", idx))
		}
		if !bytes.Equal(sigHashes[idx], sigHash) {
			return 0, errors.New(fmt.Sprintf("invalid signature hash for input %d", idx))
		}

		inputsSum += bitcoin.ToAmount(unspent.Amount, bitcoin.Decimals).Int64()
	}

	return inputsSum, nil
}

func (m *ConsensusMechanism) validateChange(tx *wire.MsgTx, inputs map[bitcoin.OutPoint]btcjson.ListUnspentResult, inputsSum, outputsSum int64) error {
	actualFee := inputsSum - outputsSum
	if actualFee <= 0 {
		return errors.New("invalid change amount")
	}

	mockedTx, err := m.client.MockTransaction(tx, inputs)
	if err != nil {
		return errors.Wrap(err, "failed to mock transaction")
	}

	var (
		targetFeeRate = float64(m.params.FeeRate) // sat/byte
		feeTolerance  = 0.1 * targetFeeRate       // 10%
		estimatedSize = mockedTx.SerializeSize()
		actualFeeRate = float64(actualFee) / float64(estimatedSize)
	)

	if math.Abs(actualFeeRate-targetFeeRate) > feeTolerance {
		return errors.New(fmt.Sprintf("provided fee rate %f is not within %f of target %f", actualFeeRate, feeTolerance, targetFeeRate))
	}

	return nil
}
