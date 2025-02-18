package withdrawal

import (
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
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
}

func (c *BitcoinWithdrawalConstructor) FromPayload(payload *anypb.Any) (BitcoinWithdrawalData, error) {
	proposalData := &p2p.BitcoinProposalData{}
	if err := payload.UnmarshalTo(proposalData); err != nil {
		return BitcoinWithdrawalData{}, errors.Wrap(err, "failed to unmarshal proposal data")
	}

	return BitcoinWithdrawalData{ProposalData: proposalData}, nil
}

func (c *BitcoinWithdrawalConstructor) FormSigningData(deposit db.Deposit) (BitcoinWithdrawalData, error) {
	// TODO: fundrawtx

	// TODO: form sighashes

	return BitcoinWithdrawalData{}, nil
}
