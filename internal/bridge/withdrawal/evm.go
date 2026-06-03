package withdrawal

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var _ DepositSigningData = EvmWithdrawalData{}
var _ Constructor[EvmWithdrawalData] = &EvmWithdrawalConstructor{}

type EvmWithdrawalData struct {
	ProposalData     *p2p.EvmProposalData
	SignedWithdrawal string
}

func (e EvmWithdrawalData) DepositIdentifiers() []db.DepositIdentifier {
	if e.ProposalData == nil || e.ProposalData.DepositId == nil {
		return nil
	}

	identifier := db.DepositIdentifier{
		ChainId: e.ProposalData.DepositId.ChainId,
		TxHash:  e.ProposalData.DepositId.TxHash,
		TxNonce: e.ProposalData.DepositId.TxNonce,
	}

	return []db.DepositIdentifier{identifier}
}

func (e EvmWithdrawalData) HashString() string {
	if e.ProposalData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(e.ProposalData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func NewEvmConstructor(client *evm.Client) *EvmWithdrawalConstructor {
	return &EvmWithdrawalConstructor{
		client: client,
	}
}

type EvmWithdrawalConstructor struct {
	client *evm.Client
}

func (c *EvmWithdrawalConstructor) FormSigningData(deposits ...db.Deposit) (*EvmWithdrawalData, error) {
	if len(deposits) == 0 {
		return nil, errors.New("invalid data: no deposits provided")
	}
	deposit := deposits[0] // Expecting only one deposit to process

	sigHash, err := c.client.GetSignHash(deposit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signing hash")
	}

	return &EvmWithdrawalData{
		ProposalData: &p2p.EvmProposalData{
			DepositId: &types.DepositIdentifier{
				ChainId: deposit.ChainId,
				TxHash:  deposit.TxHash,
				TxNonce: deposit.TxNonce,
			},
			SigData: sigHash,
		},
	}, nil
}

func (c *EvmWithdrawalConstructor) IsValid(data EvmWithdrawalData, deposits ...db.Deposit) (bool, error) {
	if len(deposits) == 0 {
		return false, errors.New("invalid data: no deposits provided")
	}
	deposit := deposits[0]

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
