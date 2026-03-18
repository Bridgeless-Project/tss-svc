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
	"github.com/Bridgeless-Project/tss-svc/pkg/merkle"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

	leaves, err := c.client.GetSignHashes(deposits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signing hashes")
	}

	tree, err := merkle.BuildTree(leaves)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build merkle tree")
	}

	root := tree.GetRoot()

	prefixedRoot := operations.SetSignaturePrefix(root)
	var depositIds []*types.DepositIdentifier
	var allProofs []*p2p.MerkleProof

	for i := range deposits {

		depositIds = append(depositIds, &types.DepositIdentifier{
			ChainId: deposits[i].ChainId,
			TxHash:  deposits[i].TxHash,
			TxNonce: deposits[i].TxNonce,
		})

		byteProof, err := tree.GetProof(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get merkle proof for leaf %d", i)
		}
		proof := make([]string, len(byteProof)+1)
		proof[0] = hexutil.Encode(tree.Leaves[i].Hash)
		for j, hash := range byteProof {
			proof[j+1] = hexutil.Encode(hash)
		}

		allProofs = append(allProofs, &p2p.MerkleProof{
			Hashes: proof,
		})
	}

	return &EvmWithdrawalData{
		ProposalData: &p2p.EvmProposalData{
			DepositIds:   depositIds,
			SigData:      prefixedRoot,
			MerkleProofs: allProofs,
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
	leaves, err := c.client.GetSignHashes(deposits)
	if err != nil {
		return false, errors.Wrap(err, "failed to get signing hashes")
	}

	tree, err := merkle.BuildTree(leaves)
	if err != nil {
		return false, errors.Wrap(err, "failed to build merkle tree")
	}

	root := tree.GetRoot()

	prefixedRoot := operations.SetSignaturePrefix(root)

	if !bytes.Equal(data.ProposalData.SigData, prefixedRoot) {
		return false, errors.New("sig data does not match the expected one")
	}

	for i := range tree.Leaves {
		proof, err := tree.GetProof(i)
		if err != nil {
			return false, errors.Wrapf(err, "failed to get proof for leaf %d", i)
		}
		expectedHashes := data.ProposalData.MerkleProofs[i].Hashes
		if hexutil.Encode(tree.Leaves[i].Hash) != expectedHashes[0] {
			return false, errors.Errorf("leaf does not match the expected one")
		}

		for j, hash := range proof {
			if hexutil.Encode(hash) != expectedHashes[j+1] {
				return false, errors.Errorf("merkle proof mismatch at leaf %d, hash %d", i, j)
			}
		}
	}
	return true, nil
}
