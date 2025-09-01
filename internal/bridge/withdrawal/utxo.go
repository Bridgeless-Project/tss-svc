package withdrawal

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
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
	identifier.TxNonce = e.ProposalData.DepositId.TxNonce

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
	client  client.Client
	helper  helper.UtxoHelper
	tssAddr string
}

func NewUtxoConstructor(client client.Client, tssPub *ecdsa.PublicKey) *UtxoWithdrawalConstructor {
	hlp := client.UtxoHelper()
	return &UtxoWithdrawalConstructor{
		client:  client,
		helper:  hlp,
		tssAddr: hlp.P2pkhAddress(tssPub),
	}
}

func (c *UtxoWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*UtxoWithdrawalData, error) {
	amount, set := new(big.Int).SetString(deposit.WithdrawalAmount, 10)
	if !set {
		return nil, errors.New("failed to parse amount")
	}
	receiverScript, err := c.helper.PayToAddrScript(deposit.Receiver)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create script")
	}
	receiverOutput := wire.NewTxOut(amount.Int64(), receiverScript)

	unspent, err := c.client.ListUnspent()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available UTXOs")
	}
	feeRate := c.client.EstimateFeeOrDefault()

	unsignedTxData, err := c.helper.NewUnsignedTransaction(
		unspent,
		feeRate,
		[]*wire.TxOut{receiverOutput},
		c.tssAddr,
	)
	if err != nil {
		// TODO: check not enough funds err
		return nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

	unsignedTxData = c.subtractFeeFromWithdrawal(unsignedTxData, feeRate)

	sigHashes, err := c.formSignatureHashes(unsignedTxData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to form signature hashes")
	}

	var buf bytes.Buffer
	if err = unsignedTxData.Tx.Serialize(&buf); err != nil {
		return nil, errors.Wrap(err, "failed to serialize transaction")
	}
	txSerialized := buf.Bytes()

	return &UtxoWithdrawalData{
		ProposalData: &p2p.BitcoinProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxNonce: deposit.TxNonce,
				TxHash:  deposit.TxHash,
			},
			SerializedTx: txSerialized,
			FeeRate:      int64(feeRate),
			SigData:      sigHashes,
		},
	}, nil
}

func (c *UtxoWithdrawalConstructor) IsValid(data UtxoWithdrawalData, deposit db.Deposit) (bool, error) {
	tx := wire.MsgTx{}
	if err := tx.Deserialize(bytes.NewReader(data.ProposalData.SerializedTx)); err != nil {
		return false, errors.Wrap(err, "failed to deserialize transaction")
	}

	feeRate := btcutil.Amount(data.ProposalData.FeeRate)
	if !utils.FeeRateValid(feeRate) {
		return false, errors.Errorf("invalid fee rate: %d", data.ProposalData.FeeRate)
	}

	unspent, err := c.client.ListUnspent()
	if err != nil {
		// TODO: RPC err
		return false, errors.Wrap(err, "failed to get available UTXOs")
	}
	usedInputs, err := utils.FindUsedInputs(tx, unspent)
	if err != nil {
		return false, errors.Wrap(err, "failed to find tx used inputs")
	}

	amount, set := new(big.Int).SetString(deposit.WithdrawalAmount, 10)
	if !set {
		return false, errors.New("failed to parse amount")
	}
	receiverScript, err := c.helper.PayToAddrScript(deposit.Receiver)
	if err != nil {
		return false, errors.Wrap(err, "failed to create script")
	}
	receiverOutput := wire.NewTxOut(amount.Int64(), receiverScript)

	unsignedTxData, err := c.helper.NewUnsignedTransaction(
		usedInputs,
		feeRate,
		[]*wire.TxOut{receiverOutput},
		c.tssAddr,
	)
	if err != nil {
		return false, errors.Wrap(err, "failed to create unsigned transaction")
	}

	unsignedTxData = c.subtractFeeFromWithdrawal(unsignedTxData, feeRate)

	var buf bytes.Buffer
	if err = unsignedTxData.Tx.Serialize(&buf); err != nil {
		return false, errors.Wrap(err, "failed to serialize transaction")
	}
	txSerialized := buf.Bytes()
	if !bytes.Equal(txSerialized, data.ProposalData.SerializedTx) {
		return false, errors.New("provided transaction does not match the expected one")
	}

	sigHashes, err := c.formSignatureHashes(unsignedTxData)
	if err != nil {
		return false, errors.Wrap(err, "failed to form signature hashes")
	}
	if len(sigHashes) != len(data.ProposalData.SigData) {
		return false, errors.New("signature hashes number mismatch")
	}
	for i, sigHash := range data.ProposalData.SigData {
		if !bytes.Equal(sigHash, sigHashes[i]) {
			return false, errors.Errorf("signature hash mismatch at index %d", i)
		}
	}

	return true, nil
}

func (c *UtxoWithdrawalConstructor) formSignatureHashes(unsignedTxData *txauthor.AuthoredTx) ([][]byte, error) {
	sigHashes := make([][]byte, len(unsignedTxData.Tx.TxIn))
	for i := range unsignedTxData.PrevScripts {
		sigHash, err := c.helper.CalculateSignatureHash(
			unsignedTxData.PrevScripts[i],
			unsignedTxData.Tx,
			i,
			int64(unsignedTxData.PrevInputValues[i]),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to calculate signature hash for tx %d", i)
		}

		sigHashes[i] = sigHash
	}

	return sigHashes, nil
}

func (c *UtxoWithdrawalConstructor) subtractFeeFromWithdrawal(tx *txauthor.AuthoredTx, feeRate btcutil.Amount) *txauthor.AuthoredTx {
	fee := c.helper.EstimateFee(tx.Tx, feeRate)
	withdrawalAmount := tx.Tx.TxOut[0].Value - int64(fee)

	if !c.client.WithdrawalAmountValid(big.NewInt(withdrawalAmount)) {
		// cannot subtract fee from withdrawalAmount, it would result in an invalid amount
		// commission will be paid by the TSS service
		// should not happen if the bridging is configured correctly
		return tx
	}

	if tx.ChangeIndex != -1 {
		tx.Tx.TxOut[0].Value -= int64(fee)
		tx.Tx.TxOut[tx.ChangeIndex].Value += int64(fee)

		return tx
	}

	// try adding a change output
	txWithChange := tx.Tx.Copy()
	changeScript, _ := c.helper.PayToAddrScript(c.tssAddr)
	txWithChange.AddTxOut(wire.NewTxOut(0, changeScript))

	feeWithChange := c.helper.EstimateFee(txWithChange, feeRate)
	withdrawalAmountWithChange := tx.Tx.TxOut[0].Value - int64(feeWithChange)
	if !c.client.WithdrawalAmountValid(big.NewInt(withdrawalAmountWithChange)) {
		// commission still will be paid by the TSS service
		// should not happen if the bridging is configured correctly
		return tx
	}

	change := int64(tx.TotalInput) - withdrawalAmountWithChange - int64(feeWithChange)

	if !c.client.WithdrawalAmountValid(big.NewInt(change)) {
		// cannot add change output, just pay the old fee from the withdrawalAmount
		tx.Tx.TxOut[0].Value -= int64(fee)

		return tx
	}

	txWithChange.TxOut[0].Value -= int64(feeWithChange)
	txWithChange.TxOut[1].Value = change
	tx.Tx = txWithChange
	tx.ChangeIndex = 1

	return tx
}
