package btc

import "math/big"

func toBigint(val float64, decimals int64) *big.Int {
	bigval := new(big.Float)
	bigval.SetFloat64(val)

	coin := new(big.Float)
	coin.SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(decimals), nil))

	bigval.Mul(bigval, coin)

	result := new(big.Int)
	bigval.Int(result)

	return result
}
