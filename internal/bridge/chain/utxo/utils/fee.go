package utils

import "github.com/btcsuite/btcd/btcutil"

var (
	// minimum fee rate is 0.00001 BTC per kilobyte
	MaxFeeRateBtcPerKvb, _     = btcutil.NewAmount(0.0001)
	DefaultFeeRateBtcPerKvb, _ = btcutil.NewAmount(0.00002)
)
