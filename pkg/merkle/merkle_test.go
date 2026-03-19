package merkle

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestMerkleTree(t *testing.T) {
	type merkle struct {
		leaves           [][]byte
		expectedProofLen int
		expectedRoot     string
	}
	testCases := map[string]merkle{
		"single_leaf": {
			leaves:           [][]byte{crypto.Keccak256([]byte("a"))},
			expectedProofLen: 0,
			expectedRoot:     "0x3ac225168df54212a25c1c01fd35bebfea408fdac2e31ddd6f80a4bbf9a5f1cb",
		},
		"two_leaves": {
			leaves: [][]byte{
				crypto.Keccak256([]byte("a")),
				crypto.Keccak256([]byte("b")),
			},
			expectedProofLen: 1,
			expectedRoot:     "0x805b21d846b189efaeb0377d6bb0d201b3872a363e607c25088f025b0c6ae1f8",
		},
		"three_leaves": {
			leaves: [][]byte{
				crypto.Keccak256([]byte("a")),
				crypto.Keccak256([]byte("b")),
				crypto.Keccak256([]byte("c")),
			},
			expectedProofLen: 2,
			expectedRoot:     "0x905b17edcf8b6fb1415b32cdbab3e02c2c93f80a345de80ea2bbf9feba9f5a55",
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
			expectedRoot:     "0x6285ee0a446cefa7ccf04a810cb6264d9b00b4fb2ce00a27a9cd8d0b2ecaf42b",
		},
	}

	for name, tCase := range testCases {
		t.Run(name, func(t *testing.T) {
			tree, err := BuildTree(tCase.leaves)
			if err != nil {
				t.Fatalf("Failed to build tree: %v", err)
			}

			root := tree.GetRoot()

			if len(root) == 0 {
				t.Fatal("Root hash should not be empty")
			}

			require.Equal(t, tCase.expectedRoot, hexutil.Encode(root))

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
