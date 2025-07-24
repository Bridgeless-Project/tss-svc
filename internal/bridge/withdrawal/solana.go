package withdrawal

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/solana"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var _ DepositSigningData = SolanaWithdrawalData{}
var _ Constructor[SolanaWithdrawalData] = &SolanaWithdrawalConstructor{}

type SolanaWithdrawalData struct {
	ProposalData     *p2p.SolanaProposalData
	SignedWithdrawal string
}

func (e SolanaWithdrawalData) DepositIdentifier() db.DepositIdentifier {
	identifier := db.DepositIdentifier{}

	if e.ProposalData == nil || e.ProposalData.DepositId == nil {
		return identifier
	}

	identifier.ChainId = e.ProposalData.DepositId.ChainId
	identifier.TxHash = e.ProposalData.DepositId.TxHash
	identifier.TxNonce = int(e.ProposalData.DepositId.TxNonce)

	return identifier
}

func (e SolanaWithdrawalData) HashString() string {
	if e.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(e.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func NewSolanaConstructor(client *solana.Client) *SolanaWithdrawalConstructor {
	return &SolanaWithdrawalConstructor{
		client: client,
	}
}

type SolanaWithdrawalConstructor struct {
	client *solana.Client
}

func (c *SolanaWithdrawalConstructor) FormSigningData(deposit db.Deposit) (*SolanaWithdrawalData, error) {
	sigHash, err := c.client.GetSignHash(deposit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signing hash")
	}

	return &SolanaWithdrawalData{
		ProposalData: &p2p.SolanaProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxHash:  deposit.TxHash,
				TxNonce: uint32(deposit.TxNonce),
			},
			SigData: sigHash,
		},
	}, nil
}

func (c *SolanaWithdrawalConstructor) IsValid(data SolanaWithdrawalData, deposit db.Deposit) (bool, error) {
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
