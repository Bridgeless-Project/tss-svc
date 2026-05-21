package utils

import (
	"sort"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/figure/v3"
)

var DefaultResharingParams = ConsolidationParams{
	MaxFeeRateSatsPerKb: MaxFeeRateBtcPerKvb,
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

var DefaultConsolidationParams = ConsolidationParams{
	MaxFeeRateSatsPerKb: MaxConsolidationFeeRateBtcPerKvb,
	SetParams: []ConsolidationSetParams{
		{
			LowerBound:     btcutil.Amount(1_000),
			UpperBound:     btcutil.Amount(100_000),
			Threshold:      30,
			MaxInputsCount: 20,
			OutsCount:      5,
		},
		{
			LowerBound:     btcutil.Amount(100_001),
			UpperBound:     btcutil.Amount(1_000_000),
			Threshold:      30,
			MaxInputsCount: 20,
			OutsCount:      5,
		},
		{
			LowerBound:     btcutil.Amount(1_000_001),
			UpperBound:     btcutil.Amount(10_000_000),
			Threshold:      30,
			MaxInputsCount: 20,
			OutsCount:      5,
		},
		{
			LowerBound:     btcutil.Amount(10_000_001),
			UpperBound:     btcutil.Amount(100_000_000),
			Threshold:      30,
			MaxInputsCount: 20,
			OutsCount:      5,
		},
		{
			LowerBound:     btcutil.Amount(100_000_001),
			UpperBound:     btcutil.Amount(100_000_000_000_000),
			Threshold:      30,
			MaxInputsCount: 20,
			OutsCount:      5,
		},
	},
}

var _ figure.Validatable = ConsolidationParams{}

type ConsolidationParams struct {
	SetParams           []ConsolidationSetParams `fig:"sets"`
	MaxFeeRateSatsPerKb btcutil.Amount           `fig:"max_fee_rate"`
}

func (c ConsolidationParams) Validate() error {
	if !FeeRateValid(c.MaxFeeRateSatsPerKb) {
		return errors.Errorf("max fee rate %s is invalid", c.MaxFeeRateSatsPerKb)
	}

	for i, set := range c.SetParams {
		if err := set.Validate(); err != nil {
			return errors.Wrapf(err, "consolidation set %d is invalid", i)
		}
	}

	return nil
}

type ConsolidationSetParams struct {
	LowerBound     btcutil.Amount `fig:"lower_bound"`      // inclusive
	UpperBound     btcutil.Amount `fig:"upper_bound"`      // inclusive
	Threshold      uint           `fig:"threshold"`        // number of inputs to trigger consolidation
	MaxInputsCount uint           `fig:"max_inputs_count"` // max number of inputs to use in consolidation tx
	OutsCount      uint           `fig:"outs_count"`       // number of inputs and outputs in consolidation tx
}

func (p ConsolidationSetParams) Validate() error {
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
				break
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
