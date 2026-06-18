package merkle

import (
	"bytes"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

type Node struct {
	Hash  []byte
	Right *Node
	Left  *Node
}

type Tree struct {
	Root   *Node
	Leaves []*Node
}

func (t *Tree) GetRoot() []byte {
	return t.Root.Hash
}

func BuildTree(data [][]byte) (*Tree, error) {
	if len(data) == 0 {
		return nil, errors.New("invalid proposal data")
	}

	leaves := make([]*Node, len(data))
	for i, hash := range data {
		leaves[i] = &Node{
			Hash: hash,
		}
	}

	currentLevel := leaves
	for len(currentLevel) > 1 {
		currentLevel = buildlevel(currentLevel)
	}

	return &Tree{
		Root:   currentLevel[0],
		Leaves: leaves,
	}, nil
}

func (t *Tree) GetProof(index int) ([][]byte, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, errors.New("invalid leaf index")
	}

	var proof [][]byte

	var currentNode = index
	var currentLevel = t.Leaves
	for len(currentLevel) > 1 {
		var siblingIndex int

		if currentNode%2 == 0 {
			siblingIndex = currentNode + 1
			if siblingIndex == len(currentLevel) {
				siblingIndex = currentNode
			}
		} else {
			siblingIndex = currentNode - 1
		}

		proof = append(proof, currentLevel[siblingIndex].Hash)

		currentLevel = buildlevel(currentLevel)

		currentNode = currentNode / 2
	}

	return proof, nil
}

func VerifyProof(rootHash []byte, leafHash []byte, proof [][]byte) bool {
	computedHash := leafHash

	for _, proofElement := range proof {
		if bytes.Compare(computedHash, proofElement) <= 0 {
			computedHash = crypto.Keccak256(append(computedHash, proofElement...))
		} else {
			computedHash = crypto.Keccak256(append(proofElement, computedHash...))
		}
	}

	return bytes.Equal(computedHash, rootHash)
}

func buildlevel(currentLevel []*Node) []*Node {
	nextLevel := make([]*Node, 0, (len(currentLevel)+1)/2)
	for i := 0; i < len(currentLevel); i += 2 {
		if i+1 == len(currentLevel) {
			combined := merge(currentLevel[i], currentLevel[i])
			nextLevel = append(nextLevel, combined)
		} else {
			parent := merge(currentLevel[i], currentLevel[i+1])
			nextLevel = append(nextLevel, parent)
		}
	}

	currentLevel = nextLevel

	return currentLevel
}

func merge(a, b *Node) *Node {
	if bytes.Compare(a.Hash, b.Hash) <= 0 {
		return &Node{
			Hash:  crypto.Keccak256(append(a.Hash, b.Hash...)),
			Left:  a,
			Right: b,
		}
	}
	return &Node{
		Hash:  crypto.Keccak256(append(b.Hash, a.Hash...)),
		Left:  b,
		Right: a,
	}
}
