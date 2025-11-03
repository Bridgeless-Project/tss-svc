package utils

import (
	"sort"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/pkg/errors"
)

var DefaultResharingParams = ConsolidationParams{
	MaxFeeRateSatsPerKb: DefaultFeeRateBtcPerKvb * 2,
	SetParams: []ConsolidationSetParams{
		{
			LowerBound:     btcutil.Amount(1000),
			UpperBound:     btcutil.Amount(100_000_000_000_000),
			MaxInputsCount: 100,
			OutsCount:      20,
			Threshold:      1,
		},
	},
}

// FIXME: adjust consolidation params
var DefaultConsolidationParams = ConsolidationParams{
	MaxFeeRateSatsPerKb: MaxConsolidationFeeRateBtcPerKvb,
	SetParams: []ConsolidationSetParams{
		{
			LowerBound:     btcutil.Amount(2000),
			UpperBound:     btcutil.Amount(50000),
			Threshold:      20,
			MaxInputsCount: 15,
			OutsCount:      1,
		},
		{
			LowerBound:     btcutil.Amount(50001),
			UpperBound:     btcutil.Amount(200000),
			Threshold:      15,
			MaxInputsCount: 10,
			OutsCount:      1,
		},
		{
			LowerBound: btcutil.Amount(200001),
			UpperBound: btcutil.Amount(1000000),
		},
	},
}

type ConsolidationParams struct {
	SetParams           []ConsolidationSetParams
	MaxFeeRateSatsPerKb btcutil.Amount
}

type ConsolidationSetParams struct {
	LowerBound     btcutil.Amount // inclusive
	UpperBound     btcutil.Amount // inclusive
	Threshold      uint           // number of inputs to trigger consolidation
	MaxInputsCount uint
	OutsCount      uint // number of inputs and outputs in consolidation tx
}

func (p *ConsolidationSetParams) Validate() error {
	if p.UpperBound == 0 {
		return errors.New("upper bound must be greater than 0")
	}
	if p.LowerBound >= p.UpperBound {
		return errors.New("lower bound must be less than upper bound")
	}

	if p.MaxInputsCount == 0 {
		return errors.New("max inputs count must be greater than 0")
	}

	if p.Threshold == 0 {
		return errors.New("threshold must be greater than 0")
	}
	if p.OutsCount == 0 {
		return errors.New("outs count must be greater than 0")
	}

	return nil
}

type ConsolidationSet struct {
	Params         ConsolidationSetParams
	outputs        []btcjson.ListUnspentResult
	outputArranger OutputArranger
}

func NewConsolidationSet(params ConsolidationSetParams) *ConsolidationSet {
	return &ConsolidationSet{Params: params, outputArranger: OldestFirstOutputArranger{}}
}

func (s *ConsolidationSet) OutputSuitable(out btcjson.ListUnspentResult) bool {
	amt, _ := btcutil.NewAmount(out.Amount)

	return amt >= s.Params.LowerBound && amt <= s.Params.UpperBound
}

func (s *ConsolidationSet) AddOutput(out btcjson.ListUnspentResult) {
	s.outputs = append(s.outputs, out)
}

func (s *ConsolidationSet) ThresholdReached() bool {
	return len(s.outputs) >= int(s.Params.Threshold)
}

func (s *ConsolidationSet) Select() []btcjson.ListUnspentResult {
	outs := s.outputArranger.ArrangeOutputs(s.outputs)

	if len(outs) <= int(s.Params.MaxInputsCount) {
		return outs
	}

	return outs[:s.Params.MaxInputsCount]
}

type ConsolidationSets []*ConsolidationSet

func (s ConsolidationSets) Len() int { return len(s) }
func (s ConsolidationSets) Less(i, j int) bool {
	return s[i].Params.LowerBound < s[j].Params.LowerBound
}
func (s ConsolidationSets) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type ConsolidationSelector struct {
	params []ConsolidationSetParams
}

func NewConsolidationSelector(params []ConsolidationSetParams) *ConsolidationSelector {
	return &ConsolidationSelector{params: params}
}

func (c *ConsolidationSelector) SelectConsolidationSet(outs []btcjson.ListUnspentResult) *ConsolidationSet {
	sets := make(ConsolidationSets, len(c.params))
	for i := range c.params {
		sets[i] = NewConsolidationSet(c.params[i])
	}

	// ensure deterministic order
	sort.Stable(sets)

	for _, out := range outs {
		for i := range sets {
			if sets[i].OutputSuitable(out) {
				sets[i].AddOutput(out)
			}
		}
	}

	for _, set := range sets {
		if set.ThresholdReached() {
			return set
		}
	}

	return nil
}
