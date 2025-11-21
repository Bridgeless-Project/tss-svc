package zano

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	zanoSdk "github.com/Bridgeless-Project/tss-svc/pkg/zano"
	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	_ consensus.SigningData            = SigningData{}
	_ consensus.Mechanism[SigningData] = &ConsensusMechanism{}
)

type SigningData struct {
	ProposalData *p2p.ZanoResharingProposalData
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
	assetId     string
	ownerPubKey string
	isEthKey    bool
	client      *zano.Client
}

func NewConsensusMechanism(
	assetId string,
	ownerPubKey string,
	isEthKey bool,
	client *zano.Client,
) *ConsensusMechanism {
	return &ConsensusMechanism{
		assetId:     assetId,
		ownerPubKey: ownerPubKey,
		isEthKey:    isEthKey,
		client:      client,
	}
}

func (c ConsensusMechanism) FormProposalData() (*SigningData, error) {
	tx, err := c.client.TransferAssetOwnershipUnsigned(c.assetId, c.ownerPubKey, c.isEthKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to form ownership unsigned transaction")
	}

	return &SigningData{
		ProposalData: &p2p.ZanoResharingProposalData{
			AssetId:        c.assetId,
			OwnerEthPubKey: c.ownerPubKey,

			OutputsAddresses: tx.DataForExternalSigning.OutputsAddresses,
			UnsignedTx:       tx.DataForExternalSigning.UnsignedTx,
			FinalizedTx:      tx.DataForExternalSigning.FinalizedTx,
			TxSecretKey:      tx.DataForExternalSigning.TxSecretKey,
			TxId:             tx.TxID,

			SigData: zanoSdk.FormSigningData(tx.TxID),
		},
	}, nil
}

func (c ConsensusMechanism) VerifyProposedData(data SigningData) error {
	details, err := c.client.DecryptTxDetails(zanoTypes.DataForExternalSigning{
		OutputsAddresses: data.ProposalData.OutputsAddresses,
		UnsignedTx:       data.ProposalData.UnsignedTx,
		FinalizedTx:      data.ProposalData.FinalizedTx,
		TxSecretKey:      data.ProposalData.TxSecretKey,
	})
	if err != nil {
		return errors.Wrap(err, "failed to decrypt tx details")
	}

	assetInfo, err := details.GetAssetInfo()
	if err != nil {
		return errors.Wrap(err, "failed to get asset info from tx details")
	}
	if err = c.validateAssetInfo(assetInfo); err != nil {
		return errors.Wrap(err, "asset info validation failed")
	}

	if !bytes.Equal(data.ProposalData.SigData, zanoSdk.FormSigningData(details.VerifiedTxID)) {
		return errors.New("sign data does not match the expected one")
	}

	return nil
}

func (c ConsensusMechanism) validateAssetInfo(assetInfo *zanoTypes.AssetDescriptorBase) error {
	if !assetInfo.IsValidTransferAssetOwnershipOperation() {
		return errors.New("invalid transfer asset ownership operation")
	}

	if *assetInfo.OptAssetId != c.assetId {
		return errors.New(
			fmt.Sprintf("asset id mismatch: expected %s, got %s", c.assetId, *assetInfo.OptAssetId),
		)
	}

	proposedOwner := assetInfo.OptDescriptor.Owner
	if c.isEthKey {
		proposedOwner = assetInfo.OptDescriptor.OwnerEthPubKey
	}

	if proposedOwner != c.ownerPubKey {
		return errors.New(
			fmt.Sprintf(
				"new owner pub key mismatch: expected %s, got %s",
				c.ownerPubKey,
				proposedOwner,
			),
		)
	}

	return nil
}
