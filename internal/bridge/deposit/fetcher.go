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

	srcInfo, dstInfo, err := p.GetTokens(identifier.ChainId, depositData.TokenAddress, depositData.DestinationChainId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token info")
	}

	withdrawalAmount, commission, err := p.GetWithdrawalAmount(depositData.DepositAmount, srcInfo, dstInfo)
	if err != nil {
		return nil, errors.Wrap(chain.ErrInvalidDepositedAmount, err.Error())
	}

	deposit := depositData.ToNewDeposit(
		withdrawalAmount,
		commission,
		dstInfo.Address,
		dstInfo.IsWrapped,
	)

	return &deposit, nil
}

func (p *Fetcher) GetTokens(
	srcChainId string,
	srcTokenAddress string,
	dstChainId string,
) (
	srcInfo, dstInfo *bridgetypes.TokenInfo,
	err error,
) {
	srcInfo, err = p.core.GetTokenInfo(srcChainId, srcTokenAddress)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get source token info")
	}
	token, err := p.core.GetToken(srcInfo.TokenId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get source token")
	}

	for _, info := range token.Info {
		if info.ChainId == dstChainId {
			return srcInfo, &info, nil
		}
	}

	return nil, nil, core.ErrDestinationTokenInfoNotFound
}

func (p *Fetcher) GetWithdrawalAmount(depositAmount *big.Int, srcInfo, dstInfo *bridgetypes.TokenInfo) (*big.Int, *big.Int, error) {
	withdrawalAmount := transformAmount(depositAmount, srcInfo.Decimals, dstInfo.Decimals)

	commissionAmount, err := bridgetypes.GetCommissionAmount(withdrawalAmount, dstInfo.CommissionRate)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get commission amount")
	}
	finalWithdrawalAmount := new(big.Int).Sub(withdrawalAmount, commissionAmount)

	minWithdrawalAmount, set := new(big.Int).SetString(dstInfo.MinWithdrawalAmount, 10)
	if !set {
		minWithdrawalAmount = big.NewInt(0)
	}

	if finalWithdrawalAmount.Cmp(minWithdrawalAmount) < 0 {
		return nil, nil, errors.New("withdrawal amount is less than minimum withdrawal amount")
	}

	return finalWithdrawalAmount, commissionAmount, nil
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
