package merkle

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestMerkleTree(t *testing.T) {
	type merkle struct {
		leaves           [][]byte
		expectedProofLen int
	}
	testCases := map[string]merkle{
		"single_leaf": {
			leaves:           [][]byte{crypto.Keccak256([]byte("a"))},
			expectedProofLen: 0,
		},
		"two_leaves": {
			leaves: [][]byte{
				crypto.Keccak256([]byte("a")),
				crypto.Keccak256([]byte("b")),
			},
			expectedProofLen: 1,
		},
		"three_leaves": {
			leaves: [][]byte{
				crypto.Keccak256([]byte("a")),
				crypto.Keccak256([]byte("b")),
				crypto.Keccak256([]byte("c")),
			},
			expectedProofLen: 2,
		},
		"ten_leaves": {
			leaves: [][]byte{
				crypto.Keccak256([]byte("a")),
				crypto.Keccak256([]byte("b")),
				crypto.Keccak256([]byte("c")),
				crypto.Keccak256([]byte("d")),
				crypto.Keccak256([]byte("e")),
				crypto.Keccak256([]byte("f")),
				crypto.Keccak256([]byte("g")),
				crypto.Keccak256([]byte("h")),
				crypto.Keccak256([]byte("i")),
				crypto.Keccak256([]byte("j")),
			},
			expectedProofLen: 4,
		},
	}

	for name, tCase := range testCases {
		t.Run(name, func(t *testing.T) {
			tree, err := BuildTree(tCase.leaves)
			if err != nil {
				t.Fatalf("Failed to build tree: %v", err)
			}

			root, err := tree.GetRootHash()
			if err != nil {
				t.Fatalf("Failed to get root hash: %v", err)
			}

			if len(root) == 0 {
				t.Fatal("Root hash should not be empty")
			}

			for i, leaf := range tCase.leaves {
				proof, err := tree.GetProof(i)

				require.NoError(t, err, "Failed to get proof for leaf %d", i)
				require.Len(t, proof, tCase.expectedProofLen, "Incorrect proof length for leaf %d", i)

				isValid := VerifyProof(root, leaf, proof)
				require.True(t, isValid, "Proof verification failed for leaf %d", i)
			}
		})
	}
}
