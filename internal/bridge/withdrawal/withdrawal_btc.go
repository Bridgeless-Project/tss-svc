package withdrawal

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/bitcoin"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

var _ DepositSigningData = BitcoinWithdrawalData{}

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

func (e BitcoinWithdrawalData) ToPayload() *anypb.Any {
	pb, _ := anypb.New(e.ProposalData)

	return pb
}

type BitcoinWithdrawalConstructor struct {
	client *bitcoin.Client
	tssPkh *btcutil.AddressPubKeyHash
}

func NewBitcoinConstructor(client *bitcoin.Client, tssPub *ecdsa.PublicKey) *BitcoinWithdrawalConstructor {
	tssPkh, err := bitcoin.PubKeyToPkhCompressed(tssPub, client.ChainParams())
	if err != nil {
		panic(fmt.Sprintf("failed to create TSS public key hash: %v", err))
	}

	return &BitcoinWithdrawalConstructor{client: client, tssPkh: tssPkh}
}

func (c *BitcoinWithdrawalConstructor) FromPayload(payload *anypb.Any) (BitcoinWithdrawalData, error) {
	proposalData := &p2p.BitcoinProposalData{}
	if err := payload.UnmarshalTo(proposalData); err != nil {
		return BitcoinWithdrawalData{}, errors.Wrap(err, "failed to unmarshal proposal data")
	}

	return BitcoinWithdrawalData{ProposalData: proposalData}, nil
}

func (c *BitcoinWithdrawalConstructor) FormSigningData(deposit db.Deposit) (BitcoinWithdrawalData, error) {
	tx, sigHashes, err := c.client.CreateUnsignedWithdrawalTx(deposit, c.tssPkh.EncodeAddress())
	if err != nil {
		return BitcoinWithdrawalData{}, errors.Wrap(err, "failed to create unsigned transaction")
	}

	var buf bytes.Buffer
	if err = tx.Serialize(&buf); err != nil {
		return BitcoinWithdrawalData{}, errors.Wrap(err, "failed to serialize transaction")
	}

	return BitcoinWithdrawalData{
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

	// validating inputs
	inputsSum, err := c.validateInputs(&tx, data.ProposalData.SigData, deposit)
	if err != nil {
		return false, errors.Wrap(err, "failed to validate inputs")
	}

	// validating fees
	if err = c.validateChange(&tx, inputsSum, outputsSum); err != nil {
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
		withdrawalAmount, ok := new(big.Int).SetString(*deposit.WithdrawalAmount, 10)
		if !ok || receiverOutput.Value != withdrawalAmount.Int64() {
			return 0, errors.New("invalid withdrawal amount")
		}
		outputsSum += receiverOutput.Value

		outAddr, err := btcutil.DecodeAddress(*deposit.Receiver, c.client.ChainParams())
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

func (c *BitcoinWithdrawalConstructor) validateInputs(tx *wire.MsgTx, sigHashes [][]byte, deposit db.Deposit) (int64, error) {
	unspent, err := c.client.ListUnspent()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get available UTXOs")
	}

	inputsSum := int64(0)
	usedInputs := make(map[string]struct{})
	for idx, inp := range tx.TxIn {
		if inp == nil {
			return 0, errors.New("nil input in transaction")
		}

		for _, u := range unspent {
			if u.TxID != inp.PreviousOutPoint.Hash.String() || u.Vout != inp.PreviousOutPoint.Index {
				continue
			}

			if _, ok := usedInputs[u.TxID]; ok {
				return 0, errors.New("double spending detected")
			}
			usedInputs[u.TxID] = struct{}{}

			scriptDecoded, err := hex.DecodeString(u.ScriptPubKey)
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

			inputsSum += bitcoin.ToAmount(u.Amount, bitcoin.Decimals).Int64()
			break
		}
	}
	if len(usedInputs) != len(tx.TxIn) {
		return 0, errors.New("not all inputs were found")
	}

	return inputsSum, nil
}

func (c *BitcoinWithdrawalConstructor) validateChange(tx *wire.MsgTx, inputsSum, outputsSum int64) error {
	change := inputsSum - outputsSum
	if change <= 0 {
		return errors.New("invalid change amount")
	}

	// TODO: add change validation

	return nil
}
