package evm

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/ethereum/go-ethereum/crypto"
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
		operation, err = operations.NewWithdrawNativeContent(data)
	} else {
		operation, err = operations.NewWithdrawERC20Content(data)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create operation")
	}

	hash := operation.CalculateHash()
	prefixedHash := operations.SetSignaturePrefix(hash)

	return prefixedHash, nil
}

func (p *Client) Sign(data db.Deposit) ([]byte, error) {
	if !p.chain.Meta.Centralized {
		return nil, errors.New("signing is only supported for centralized chains")
	}

	signHash, err := p.GetSignHash(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to form withdrawal signing hash")
	}

	signature, err := crypto.Sign(signHash, p.chain.Meta.SignerKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign withdrawal")
	}

	signature[64] += 27

	return signature, nil
}
