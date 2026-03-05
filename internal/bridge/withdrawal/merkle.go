package withdrawal

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

func (t *Tree) GetRootHash() ([]byte, error) {
	if t == nil || t.Root == nil {
		return nil, errors.New("failed to get merkle tree root")
	}

	return t.Root.Hash, nil
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
	}

	return &Tree{
		Root:   currentLevel[0],
		Leaves: leaves,
	}, nil
}

func merge(left, right *Node) *Node {
	var parentHash []byte

	if bytes.Compare(left.Hash, right.Hash) <= 0 {
		parentHash = crypto.Keccak256(append(left.Hash, right.Hash...))
	} else {
		parentHash = crypto.Keccak256(append(right.Hash, left.Hash...))
	}

	return &Node{
		Hash:  parentHash,
		Left:  left,
		Right: right,
	}
}
