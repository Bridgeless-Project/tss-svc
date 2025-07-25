package utils

const ConsolidationThreshold = 20

type ConsolidateOutputsParams struct {
	// satsPerKB
	FeeRate        uint64
	MaxInputsCount int
	// Threshold for the number of inputs to consolidate
	InputsThreshold int
	OutputsCount    int
}

var DefaultConsolidateOutputsParams = ConsolidateOutputsParams{
	FeeRate:         uint64(DefaultFeeRateBtcPerKvb),
	MaxInputsCount:  50,
	OutputsCount:    5,
	InputsThreshold: ConsolidationThreshold,
}
