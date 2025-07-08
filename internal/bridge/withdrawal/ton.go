package withdrawal

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/ton"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var _ DepositSigningData = TonWithdrawalData{}
var _ Constructor[TonWithdrawalData] = &TonWithdrawalConstructor{}

type TonWithdrawalData struct {
	ProposalData     *p2p.TonProposalData
	SignedWithdrawal string
}

func (e TonWithdrawalData) DepositIdentifier() db.DepositIdentifier {
	identifier := db.DepositIdentifier{}

	if e.ProposalData == nil || e.ProposalData.DepositId == nil {
		return identifier
	}

	identifier.ChainId = e.ProposalData.DepositId.ChainId
	identifier.TxHash = e.ProposalData.DepositId.TxHash
	identifier.TxNonce = int(e.ProposalData.DepositId.TxNonce)

	return identifier
}

func (e TonWithdrawalData) HashString() string {
	if e.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(e.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func NewTonConstructor(client *ton.Client) *TonWithdrawalConstructor {
	return &TonWithdrawalConstructor{
		client: client,
	}
}

type TonWithdrawalConstructor struct {
	client *ton.Client
}

func (c *TonWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*TonWithdrawalData, error) {
	sigHash, err := c.client.GetSignHash(deposit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signing hash")
	}

	return &TonWithdrawalData{
		ProposalData: &p2p.TonProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxHash:  deposit.TxHash,
				TxNonce: uint32(deposit.TxNonce),
			},
			SigData: sigHash,
		},
	}, nil
}

func (c *TonWithdrawalConstructor) IsValid(data TonWithdrawalData, deposit db.Deposit) (bool, error) {
	if data.ProposalData == nil {
		return false, errors.New("invalid proposal data")
	}

	sigHash, err := c.client.GetSignHash(deposit)
	if err != nil {
		return false, errors.Wrap(err, "failed to get signing hash")
	}

	if !bytes.Equal(data.ProposalData.SigData, sigHash) {
		return false, errors.New("sig data does not match the expected one")
	}

	return true, nil
}
