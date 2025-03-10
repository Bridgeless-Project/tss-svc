package withdrawal

import (
	"bytes"

	"github.com/hyle-team/tss-svc/internal/bridge/clients/evm"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

var _ DepositSigningData = EvmWithdrawalData{}
var _ Constructor[EvmWithdrawalData] = &EvmWithdrawalConstructor{}

type EvmWithdrawalData struct {
	ProposalData     *p2p.EvmProposalData
	SignedWithdrawal string
}

func (e EvmWithdrawalData) DepositIdentifier() db.DepositIdentifier {
	identifier := db.DepositIdentifier{}

	if e.ProposalData == nil || e.ProposalData.DepositId == nil {
		return identifier
	}

	identifier.ChainId = e.ProposalData.DepositId.ChainId
	identifier.TxHash = e.ProposalData.DepositId.TxHash
	identifier.TxNonce = int(e.ProposalData.DepositId.TxNonce)

	return identifier
}

func (e EvmWithdrawalData) ToPayload() *anypb.Any {
	pb, _ := anypb.New(e.ProposalData)

	return pb
}

func NewEvmConstructor(client *evm.Client) *EvmWithdrawalConstructor {
	return &EvmWithdrawalConstructor{
		client: client,
	}
}

type EvmWithdrawalConstructor struct {
	client *evm.Client
}

func (c *EvmWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*EvmWithdrawalData, error) {
	sigHash, err := c.client.GetSignHash(deposit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signing hash")
	}

	return &EvmWithdrawalData{
		ProposalData: &p2p.EvmProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxHash:  deposit.TxHash,
				TxNonce: uint32(deposit.TxNonce),
			},
			SigData: sigHash,
		},
	}, nil
}

func (c *EvmWithdrawalConstructor) IsValid(data EvmWithdrawalData, deposit db.Deposit) (bool, error) {
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

func (c *EvmWithdrawalConstructor) FromPayload(payload *anypb.Any) (*EvmWithdrawalData, error) {
	proposalData := &p2p.EvmProposalData{}
	if err := payload.UnmarshalTo(proposalData); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal proposal data")
	}

	return &EvmWithdrawalData{ProposalData: proposalData}, nil
}
