package deposit

import (
	"math/big"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
)

type Fetcher struct {
	core    *connector.Connector
	clients chain.Repository
}

func NewFetcher(clients chain.Repository, core *connector.Connector) *Fetcher {
	return &Fetcher{
		clients: clients,
		core:    core,
	}
}

func (p *Fetcher) FetchDeposit(identifier db.DepositIdentifier) (*db.Deposit, error) {
	sourceClient, err := p.clients.Client(identifier.ChainId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting source clients")
	}
	if !sourceClient.TransactionHashValid(identifier.TxHash) {
		return nil, errors.New("invalid transaction hash")
	}

	depositData, err := sourceClient.GetDepositData(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposit data")
	}

	dstClient, err := p.clients.Client(depositData.DestinationChainId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting destination clients")
	}
	if !dstClient.AddressValid(depositData.DestinationAddress) {
		return nil, errors.Wrap(chain.ErrInvalidReceiverAddress, depositData.DestinationAddress)
	}

	if !bridgetypes.IsDefaultReferralId(uint32(depositData.ReferralId)) {
		// check if referral id is valid
		_, err = p.core.GetReferralById(depositData.ReferralId)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get referral info")
		}
	}

	srcTokenInfo, err := p.core.GetTokenInfo(identifier.ChainId, depositData.TokenAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get source token info")
	}
	token, err := p.core.GetToken(srcTokenInfo.TokenId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get source token")
	}
	dstTokenInfo, err := getDstTokenInfo(token, depositData.DestinationChainId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dst token info")
	}

	withdrawalAmount := transformAmount(depositData.DepositAmount, srcTokenInfo.Decimals, dstTokenInfo.Decimals)
	if !dstClient.WithdrawalAmountValid(withdrawalAmount) {
		return nil, chain.ErrInvalidDepositedAmount
	}
	commissionAmount, err := bridgetypes.GetCommissionAmount(withdrawalAmount, token.CommissionRate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get commission amount")
	}
	finalWithdrawalAmount := new(big.Int).Sub(withdrawalAmount, commissionAmount)
	if !dstClient.WithdrawalAmountValid(finalWithdrawalAmount) {
		return nil, errors.Wrap(chain.ErrInvalidDepositedAmount, "invalid final withdrawal amount")
	}

	deposit := depositData.ToNewDeposit(
		finalWithdrawalAmount,
		commissionAmount,
		dstTokenInfo.Address,
		dstTokenInfo.IsWrapped,
	)

	return &deposit, nil
}

func transformAmount(amount *big.Int, currentDecimals uint64, targetDecimals uint64) *big.Int {
	result, _ := new(big.Int).SetString(amount.String(), 10)

	if currentDecimals == targetDecimals {
		return result
	}

	if currentDecimals < targetDecimals {
		for i := uint64(0); i < targetDecimals-currentDecimals; i++ {
			result.Mul(result, new(big.Int).SetInt64(10))
		}
	} else {
		for i := uint64(0); i < currentDecimals-targetDecimals; i++ {
			result.Div(result, new(big.Int).SetInt64(10))
		}
	}

	return result
}

func getDstTokenInfo(token bridgetypes.Token, dstChainId string) (bridgetypes.TokenInfo, error) {
	for _, info := range token.Info {
		if info.ChainId == dstChainId {
			return info, nil
		}
	}

	return bridgetypes.TokenInfo{}, core.ErrDestinationTokenInfoNotFound
}
