package utils

import (
	"math/rand/v2"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
)

func makeRandomOuts(count, min, max int64) []btcjson.ListUnspentResult {
	// generate count random outputs between min and max
	outs := make([]btcjson.ListUnspentResult, count)
	for i := range count {
		amount := btcutil.Amount(rand.Int64N(max-min) + min)
		outs[i] = btcjson.ListUnspentResult{
			Amount: amount.ToBTC(),
		}
	}
	return outs
}
func shuffle(s []btcjson.ListUnspentResult) {
	rand.Shuffle(len(s), func(i, j int) {
		s[i], s[j] = s[j], s[i]
	})
}

func Test_ConsolidationSelector(t *testing.T) {
	params := DefaultConsolidationParams
	selector := NewConsolidationSelector(params.SetParams)

	tc := map[string]struct {
		outs        []btcjson.ListUnspentResult
		expectedMin btcutil.Amount
		expectedMax btcutil.Amount
	}{
		"no suitable outputs": {
			outs: func() []btcjson.ListUnspentResult {
				outs := makeRandomOuts(100, 1, 500)
				outs = append(outs, makeRandomOuts(20, 1000, 100_000)...)
				outs = append(outs, makeRandomOuts(17, 100_001, 1_000_000)...)
				outs = append(outs, makeRandomOuts(15, 1_000_001, 10_000_000)...)
				outs = append(outs, makeRandomOuts(23, 10_000_001, 100_000_000)...)
				return outs
			}(),
		},
		"suitable outputs": {
			outs:        makeRandomOuts(50, 10_000, 100_000),
			expectedMin: btcutil.Amount(10_000),
			expectedMax: btcutil.Amount(100_000),
		},
		"multiple suitable outputs": {
			outs: func() []btcjson.ListUnspentResult {
				outs := makeRandomOuts(40, 100_001, 1_000_000)
				outs = append(outs, makeRandomOuts(55, 10_000_001, 100_000_000)...)
				outs = append(outs, makeRandomOuts(30, 1_000, 10_000)...)
				return outs
			}(),
			expectedMin: btcutil.Amount(1_000),
			expectedMax: btcutil.Amount(10_000),
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			shuffle(c.outs)
			set := selector.SelectConsolidationSet(c.outs)
			if set == nil {
				if c.expectedMax != 0 {
					t.Fatalf("expected a consolidation set, got nil")
				}
				return
			}
			if c.expectedMax == 0 {
				t.Fatalf("expected no consolidation set, got one")
			}

			minimum := btcutil.Amount(100_000_000_000_000)
			maximum := btcutil.Amount(0)
			for _, out := range set.outputs {
				amt, _ := btcutil.NewAmount(out.Amount)
				if amt < minimum {
					minimum = amt
				}
				if amt > maximum {
					maximum = amt
				}
			}
			if minimum < c.expectedMin || maximum > c.expectedMax {
				t.Fatalf("expected outputs between %s and %s, got between %s and %s",
					c.expectedMin.String(), c.expectedMax.String(),
					minimum.String(), maximum.String(),
				)
			}
		})
	}

}
