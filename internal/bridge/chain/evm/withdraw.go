package evm

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	operations2 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
)

type Operation interface {
	CalculateHash() []byte
}

func (p *Client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(bridge.ZeroAmount) != 1 {
		return false
	}

	return true
}

func (p *Client) GetSignHash(data db.Deposit) ([]byte, error) {
	var operation Operation
	var err error

	if data.WithdrawalToken == bridge.DefaultNativeTokenAddress {
		operation, err = operations2.NewWithdrawNativeContent(data)
	} else {
		operation, err = operations2.NewWithdrawERC20Content(data)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create operation")
	}

	hash := operation.CalculateHash()
	prefixedHash := operations2.SetSignaturePrefix(hash)

	return prefixedHash, nil
}
