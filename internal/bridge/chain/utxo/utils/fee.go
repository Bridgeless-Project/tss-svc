package utils

import (
	"math/big"

	"github.com/btcsuite/btcd/btcutil"
)

var (
	// minimum fee rate is 0.00001 BTC per kilobyte
	MaxFeeRateBtcPerKvb, _              = btcutil.NewAmount(0.00005)
	DefaultFeeRateBtcPerKvb, _          = btcutil.NewAmount(0.00001)
	MaxConsolidationFeeRateBtcPerKvb, _ = btcutil.NewAmount(0.000015)

	DustAmount = big.NewInt(547)
)

func FeeRateValid(fee btcutil.Amount) bool {
	if fee < DefaultFeeRateBtcPerKvb || fee > MaxFeeRateBtcPerKvb {
		return false
	}

	return true
}
