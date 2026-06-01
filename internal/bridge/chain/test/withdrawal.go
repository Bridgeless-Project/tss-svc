package test

import (
	"crypto/sha256"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
)

const MockSigningMessage = "frost mock string"

func (c *Client) WithdrawalAmountValid(amount *big.Int) bool {
	return amount.Cmp(bridge.ZeroAmount) == 1
}

func (c *Client) GetSignHash(_ db.Deposit) ([]byte, error) {
	hash := sha256.Sum256([]byte(MockSigningMessage))
	return hash[:], nil
}
