package withdrawal

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	zanoSdk "github.com/Bridgeless-Project/tss-svc/pkg/zano"
	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	_ DepositSigningData              = ZanoWithdrawalData{}
	_ Constructor[ZanoWithdrawalData] = &ZanoWithdrawalConstructor{}
)

type ZanoWithdrawalData struct {
	ProposalData *p2p.ZanoProposalData
}

func (z ZanoWithdrawalData) DepositIdentifier() db.DepositIdentifier {
	identifier := db.DepositIdentifier{}

	if z.ProposalData == nil || z.ProposalData.DepositId == nil {
		return identifier
	}

	identifier.ChainId = z.ProposalData.DepositId.ChainId
	identifier.TxHash = z.ProposalData.DepositId.TxHash
	identifier.TxNonce = z.ProposalData.DepositId.TxNonce

	return identifier
}

func (z ZanoWithdrawalData) HashString() string {
	if z.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(z.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type ZanoWithdrawalConstructor struct {
	client *zano.Client
}

func NewZanoConstructor(client *zano.Client) *ZanoWithdrawalConstructor {
	return &ZanoWithdrawalConstructor{
		client: client,
	}
}

func (c *ZanoWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*ZanoWithdrawalData, error) {
	tx, err := c.client.EmitAssetUnsigned(deposit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to form zano withdrawal data")
	}

	return &ZanoWithdrawalData{
		ProposalData: &p2p.ZanoProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxHash:  deposit.TxHash,
				TxNonce: deposit.TxNonce,
			},
			OutputsAddresses: tx.DataForExternalSigning.OutputsAddresses,
			UnsignedTx:       tx.DataForExternalSigning.UnsignedTx,
			FinalizedTx:      tx.DataForExternalSigning.FinalizedTx,
			TxSecretKey:      tx.DataForExternalSigning.TxSecretKey,
			TxId:             tx.TxID,
			SigData:          zanoSdk.FormSigningData(tx.TxID),
		},
	}, nil
}

func (c *ZanoWithdrawalConstructor) IsValid(data ZanoWithdrawalData, deposit db.Deposit) (bool, error) {
	details, err := c.client.DecryptTxDetails(zanoTypes.DataForExternalSigning{
		OutputsAddresses: data.ProposalData.OutputsAddresses,
		UnsignedTx:       data.ProposalData.UnsignedTx,
		FinalizedTx:      data.ProposalData.FinalizedTx,
		TxSecretKey:      data.ProposalData.TxSecretKey,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to decrypt tx details")
	}

	// validating transaction details:
	// - there should be at most one output for change
	// - other outputs should be equal to the deposit amount and pointed to the recipient

	mintedAmount := big.NewInt(0)
	changeOutputChecked := false
	for _, output := range details.DecodedOutputs {
		switch {
		case output.Address == deposit.Receiver:
			if output.AssetID == deposit.WithdrawalToken {
				mintedAmount.Add(mintedAmount, output.Amount)
			}
		default:
			// FIXME: CHECK OUTPUT RECEIVER AND CHANGE ASSET ID FOR ZANO
			if !changeOutputChecked {
				changeOutputChecked = true
			} else {
				return false, errors.New("more than one non-emit output found")
			}
		}
	}

	expectedAmount, _ := new(big.Int).SetString(deposit.WithdrawalAmount, 10)
	if mintedAmount.Cmp(expectedAmount) != 0 {
		return false, errors.New("minted amount does not match the expected one")
	}

	if !bytes.Equal(data.ProposalData.SigData, zanoSdk.FormSigningData(details.VerifiedTxID)) {
		return false, errors.New("sig data does not match the expected one")
	}

	return true, nil
}
