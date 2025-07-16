package helper

import (
	"sort"

	"github.com/btcsuite/btcd/btcjson"
)

type OutputArranger interface {
	ArrangeOutputs(outs []btcjson.ListUnspentResult) []btcjson.ListUnspentResult
}

type sortByAmount []btcjson.ListUnspentResult

func (s sortByAmount) Len() int { return len(s) }
func (s sortByAmount) Less(i, j int) bool {
	return s[i].Amount < s[j].Amount
}
func (s sortByAmount) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type LargestFirstOutputArranger struct{}

func (LargestFirstOutputArranger) ArrangeOutputs(outs []btcjson.ListUnspentResult) []btcjson.ListUnspentResult {
	sort.Sort(sort.Reverse(sortByAmount(outs)))

	return outs
}
