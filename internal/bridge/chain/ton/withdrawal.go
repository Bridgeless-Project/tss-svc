package ton

import (
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
	"math/big"
)

func (c *Client) WithdrawalAmountValid(amount *big.Int) bool {
	return amount.Cmp(bridge.ZeroAmount) == 1
}

func (c *Client) GetSignHash(deposit db.Deposit) ([]byte, error) {

	switch deposit.WithdrawalToken {

	case bridge.DefaultNativeTokenAddress:
		hash, err := c.getWithdrawalNativeHash(deposit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get withdrawal native hash")
		}
		return hash, nil
	default:
		hash, err := c.getWithdrawalJettonHash(deposit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get withdrawal jetton hash")
		}
		return hash, nil
	}
}
