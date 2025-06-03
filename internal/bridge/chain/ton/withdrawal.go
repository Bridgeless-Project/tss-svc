package ton

import (
	"github.com/hyle-team/tss-svc/internal/bridge"
	"math/big"
)

func (c *Client) WithdrawalAmountValid(amount *big.Int) bool {
	return amount.Cmp(bridge.ZeroAmount) == 1
}
