package deposit

import (
	bridgetypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"math/big"

	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/core/connector"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

const prec = 128

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
		return nil, errors.Wrap(err, "failed to get withdrawal data")
	}

	dstClient, err := p.clients.Client(depositData.DestinationChainId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting destination clients")
	}
	if !dstClient.AddressValid(depositData.DestinationAddress) {
		return nil, errors.Wrap(chain.ErrInvalidReceiverAddress, depositData.DestinationAddress)
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

	commissionAmount, err := getCommissionAmount(withdrawalAmount, token.CommissionRate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get commission amount")
	}

	deposit := depositData.ToNewDeposit(big.NewInt(0).Sub(withdrawalAmount, commissionAmount), commissionAmount,
		dstTokenInfo.Address, dstTokenInfo.IsWrapped)

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
func getCommissionAmount(withdrawalAmount *big.Int, commissionRate string) (*big.Int, error) {
	rate, ok := new(big.Float).SetString(commissionRate)
	if !ok {
		return nil, errors.New("invalid commission rate")
	}

	rate.SetPrec(prec)
	amount := new(big.Float).SetInt(withdrawalAmount).SetPrec(prec)
	commission := new(big.Float).SetPrec(prec).Mul(amount, rate)
	commission.Quo(commission, big.NewFloat(100).SetPrec(prec))

	commission.SetMode(big.ToNearestEven)

	if commission.Sign() >= 0 {
		commission.Add(commission, new(big.Float).SetPrec(prec).SetFloat64(0.5))
	} else {
		commission.Sub(commission, new(big.Float).SetPrec(prec).SetFloat64(0.5))
	}

	commissionInt, _ := commission.Int(nil)
	return commissionInt, nil
}

func getDstTokenInfo(token bridgetypes.Token, dstChainId string) (bridgetypes.TokenInfo, error) {
	for _, info := range token.Info {
		if info.ChainId == dstChainId {
			return info, nil
		}
	}

	return bridgetypes.TokenInfo{}, errors.New("dst token info not found")
}
