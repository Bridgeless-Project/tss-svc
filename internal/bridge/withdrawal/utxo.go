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
	client  client.Client
	helper  helper.UtxoHelper
	tssAddr string
}

func NewBitcoinConstructor(client client.Client, tssPub *ecdsa.PublicKey) *UtxoWithdrawalConstructor {
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

	unsignedTxData, err := c.helper.NewUnsignedTransaction(
		unspent,
		utils.DefaultFeeRateBtcPerKvb,
		[]*wire.TxOut{receiverOutput},
		c.tssAddr,
	)
	if err != nil {
		// TODO: check not enough funds err
		return nil, errors.Wrap(err, "failed to create unsigned transaction")
	}

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
				TxNonce: uint32(deposit.TxNonce),
				TxHash:  deposit.TxHash,
			},
			SerializedTx: txSerialized,
			FeeRate:      int64(utils.DefaultFeeRateBtcPerKvb),
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
	sigHashes := make([][]byte, len(unsignedTxData.PrevScripts))
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

		sigHashes = append(sigHashes, sigHash)
	}

	return sigHashes, nil
}
