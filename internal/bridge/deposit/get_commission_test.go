package deposit

import (
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func Test_GetCommissionAmount(t *testing.T) {

	type tc struct {
		withdrawalAmount *big.Int
		commissionRate   float32
		expected         *big.Int
	}

	testCases := map[string]tc{
		"should get commission amount for integer rate": {
			withdrawalAmount: big.NewInt(1000),
			commissionRate:   1,
			expected:         big.NewInt(10),
		},

		"should get commission amount for float rate": {
			withdrawalAmount: big.NewInt(1000_000_000),
			commissionRate:   0.5,
			expected:         big.NewInt(5000000),
		},
		"should get commission amount for float rate with many decimals": {
			withdrawalAmount: big.NewInt(1000_000_000),
			commissionRate:   5.32256666,
			expected:         big.NewInt(53225600),
		},
	}

	for name, tCase := range testCases {
		t.Run(name, func(t *testing.T) {
			result, _ := getCommissionAmount(tCase.withdrawalAmount, tCase.commissionRate)
			require.Equal(t, tCase.expected, result)
		})
	}

}
