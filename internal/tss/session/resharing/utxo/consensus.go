package utxo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	utxoutils "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	client     client.Client
	helper     helper.UtxoHelper
	dstAddr    string
	maxFeeRate btcutil.Amount

	selector         *utxoutils.ConsolidationSelector
	consolidationSet *utxoutils.ConsolidationSet
	feeRate          *btcutil.Amount
}

func NewConsensusMechanism(client client.Client, dst string, params utxoutils.ConsolidationParams) *ConsensusMechanism {
	helper := client.UtxoHelper()
	if _, err := helper.PayToAddrScript(dst); err != nil {
		panic(errors.Wrapf(err, "failed to create script for destination address %s", dst))
	}

	return &ConsensusMechanism{
		client,
		helper,
		dst,
		params.MaxFeeRateSatsPerKb,
		utxoutils.NewConsolidationSelector(params.SetParams),
		nil, nil,
	}
}

func (m *ConsensusMechanism) SelectConsolidationSet() (bool, error) {
	m.resetPreviousConsolidationParams()

	feeRate := m.client.EstimateFeeOrDefault()
	if feeRate > m.maxFeeRate {
		return false, nil
	}
	m.feeRate = &feeRate
	unspent, err := m.client.ListUnspent()
	if err != nil {
		return false, errors.Wrap(err, "failed to list unspent outputs")
	}

	m.consolidationSet = m.selector.SelectConsolidationSet(unspent)

	return m.consolidationSet != nil, nil
}

func (m *ConsensusMechanism) FormProposalData() (*SigningData, error) {
	if m.consolidationSet == nil || m.feeRate == nil {
		return nil, errors.New("consolidation set is not selected")
	}

	unspent := m.consolidationSet.Select()
	tx, sigHashes, err := m.consolidateOutputs(unspent)
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
			FeeRate:      int64(*m.feeRate),
			SigData:      sigHashes,
		},
	}, nil
}

func (m *ConsensusMechanism) consolidateOutputs(unspent []btcjson.ListUnspentResult) (*wire.MsgTx, [][]byte, error) {
	arranged := m.helper.ArrangeOutputs(unspent)
	receiverScript, _ := m.helper.PayToAddrScript(m.dstAddr)

	tx := wire.NewMsgTx(wire.TxVersion)

	outsCount := m.consolidationSet.Params.OutsCount
	for range outsCount {
		tx.AddTxOut(wire.NewTxOut(0, receiverScript))
	}

	totalAmount := int64(0)
	for i := range len(arranged) {
		hash, err := chainhash.NewHashFromStr(unspent[i].TxID)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to parse tx hash for input %d", i))
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(hash, unspent[i].Vout), nil, nil))
		totalAmount += utxoutils.ToUnits(unspent[i].Amount)
	}

	fees := m.helper.EstimateFee(tx, *m.feeRate)
	consolidationAmount := totalAmount - int64(fees)
	amountPerOutput := consolidationAmount / int64(outsCount)

	if !m.client.WithdrawalAmountValid(big.NewInt(amountPerOutput)) {
		return nil, nil, errors.New("amount per output is too small")
	}

	for _, out := range tx.TxOut {
		out.Value = amountPerOutput
	}
	// adding the remainder to the first output
	tx.TxOut[0].Value += consolidationAmount % int64(outsCount)

	sigHashes := make([][]byte, len(tx.TxIn))
	for i := range tx.TxIn {
		utxo := arranged[i]

		scriptDecoded, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to decode script for input %d", i))
		}
		sigHash, err := m.helper.CalculateSignatureHash(scriptDecoded, tx, i, utxoutils.ToUnits(utxo.Amount))
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to calculate signature hash for input %d", i))
		}

		sigHashes[i] = sigHash
	}

	return tx, sigHashes, nil
}

func (m *ConsensusMechanism) VerifyProposedData(data SigningData) error {
	if m.consolidationSet == nil {
		return errors.New("consolidation set was not selected")
	}
	feeRate := btcutil.Amount(data.ProposalData.FeeRate)
	if !utxoutils.FeeRateValid(feeRate) {
		return errors.Errorf("invalid fee rate %d", feeRate)
	}
	if feeRate > m.maxFeeRate {
		return errors.Errorf("fee rate %d exceeds maximum allowed %d", feeRate, m.maxFeeRate)
	}
	m.feeRate = &feeRate

	tx := wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(data.ProposalData.SerializedTx)); err != nil {
		return errors.Wrap(err, "failed to deserialize transaction")
	}

	unspent := m.consolidationSet.Select()
	originalTx, sigHashes, err := m.consolidateOutputs(unspent)
	if err != nil {
		return errors.Wrap(err, "failed to consolidate outputs from used inputs")
	}

	var buf bytes.Buffer
	if err = originalTx.Serialize(&buf); err != nil {
		return errors.Wrap(err, "failed to serialize original transaction")
	}
	if !bytes.Equal(buf.Bytes(), data.ProposalData.SerializedTx) {
		return errors.New("provided transaction does not match the expected one")
	}
	if len(sigHashes) != len(data.ProposalData.SigData) {
		return errors.New("signature hashes number mismatch")
	}
	for i := range data.ProposalData.SigData {
		if !bytes.Equal(data.ProposalData.SigData[i], sigHashes[i]) {
			return errors.Errorf("signature hash mismatch at index %d", i)
		}
	}

	return nil
}

func (m *ConsensusMechanism) resetPreviousConsolidationParams() {
	m.consolidationSet = nil
	m.feeRate = nil
}
