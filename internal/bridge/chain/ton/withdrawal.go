package ton

import (
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	"math/big"
)

func (c *Client) WithdrawalAmountValid(amount *big.Int) bool {
	return amount.Cmp(bridge.ZeroAmount) == 1
}

func (c *Client) GetSignHash(deposit db.Deposit) ([]byte, error) {
	// TODO: Implement it
	return nil, nil
}
