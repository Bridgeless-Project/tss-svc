package deposit

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TransformAmount(t *testing.T) {
	type tc struct {
		amount     *big.Int
		cDec, tDec uint64
		expected   *big.Int
	}

	testCases := map[string]tc{
		"should not change": {
			amount:   big.NewInt(100),
			cDec:     1,
			tDec:     1,
			expected: big.NewInt(100),
		},
		"should transform correctly (target is larger that current)": {
			amount:   big.NewInt(100_000_000),
			cDec:     6,
			tDec:     12,
			expected: big.NewInt(100_000_000_000_000),
		},
		"should transform correctly (target is smaller that current)": {
			amount:   big.NewInt(100_000_000_000_000_000),
			cDec:     15,
			tDec:     6,
			expected: big.NewInt(100_000_000),
		},
		"should transform correctly (target is much bigger than current)": {
			amount:   big.NewInt(100),
			cDec:     6,
			tDec:     18,
			expected: big.NewInt(100_000_000_000_000),
		},
		"should transform to zero (target is much lower that current)": {
			amount:   big.NewInt(100_000),
			cDec:     18,
			tDec:     6,
			expected: big.NewInt(0),
		},
	}

	for name, tCase := range testCases {
		t.Run(name, func(t *testing.T) {
			result := transformAmount(tCase.amount, tCase.cDec, tCase.tDec)
			require.Equal(t, tCase.expected.String(), result.String())
		})
	}
}
