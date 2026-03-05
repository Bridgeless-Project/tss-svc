package withdrawal

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
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
	var identifiers []db.DepositIdentifier

	if e.ProposalData == nil {
		return identifiers
	}

	for _, pbId := range e.ProposalData.DepositIds {
		identifiers = append(identifiers, db.DepositIdentifier{
			ChainId: pbId.ChainId,
			TxHash:  pbId.TxHash,
			TxNonce: pbId.TxNonce,
		})
	}
	return identifiers
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
	sort.Slice(deposits, func(i, j int) bool {
		return deposits[i].TxHash < deposits[j].TxHash
	})

	leaves, err := c.client.GetSignHashMerkle(deposits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signing hashes")
	}

	tree, err := BuildTree(leaves)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build merkle tree")
	}

	root, err := tree.GetRootHash()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get merkle tree root")
	}

	prefixedRoot := operations.SetSignaturePrefix(root)
	var depositIds []*types.DepositIdentifier
	for _, d := range deposits {
		depositIds = append(depositIds, &types.DepositIdentifier{
			ChainId: d.ChainId,
			TxHash:  d.TxHash,
			TxNonce: d.TxNonce,
		})
	}

	return &EvmWithdrawalData{
		ProposalData: &p2p.EvmProposalData{
			DepositIds: depositIds,
			SigData:    prefixedRoot,
		},
	}, nil
}

func (c *EvmWithdrawalConstructor) IsValid(data EvmWithdrawalData, deposits ...db.Deposit) (bool, error) {
	if data.ProposalData == nil {
		return false, errors.New("invalid proposal data")
	}

	sort.Slice(deposits, func(i, j int) bool {
		return deposits[i].TxHash < deposits[j].TxHash
	})

	leaves, err := c.client.GetSignHashMerkle(deposits)
	if err != nil {
		return false, errors.Wrap(err, "failed to get signing hashes")
	}

	tree, err := BuildTree(leaves)
	if err != nil {
		return false, errors.Wrap(err, "failed to build merkle tree")
	}

	root, err := tree.GetRootHash()
	if err != nil {
		return false, errors.Wrap(err, "failed to get merkle tree root")
	}

	prefixedRoot := operations.SetSignaturePrefix(root)

	if !bytes.Equal(data.ProposalData.SigData, prefixedRoot) {
		var hashes []string
		for _, d := range deposits {
			hashes = append(hashes, d.TxHash)
		}
		return false, errors.New("sig data does not match the expected one")
	}

	return true, nil
}
