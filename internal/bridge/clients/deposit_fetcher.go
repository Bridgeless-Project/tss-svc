package clients

import (
	"math/big"

	connector "github.com/hyle-team/tss-svc/internal/core/connector"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
)

type DepositFetcher struct {
	core    *connector.Connector
	clients ClientsRepository
}

func NewDepositFetcher(clients ClientsRepository, core *connector.Connector) *DepositFetcher {
	return &DepositFetcher{
		clients: clients,
		core:    core,
	}
}

func (p *DepositFetcher) FetchDeposit(identifier db.DepositIdentifier) (*db.Deposit, error) {
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
		return nil, errors.Wrap(ErrInvalidReceiverAddress, depositData.DestinationAddress)
	}

	srcTokenInfo, err := p.core.GetTokenInfo(identifier.ChainId, depositData.TokenAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get source token info")
	}
	dstTokenInfo, err := p.core.GetDestinationTokenInfo(identifier.ChainId, depositData.TokenAddress, depositData.DestinationChainId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get destination token info")
	}

	withdrawalAmount := transformAmount(depositData.DepositAmount, srcTokenInfo.Decimals, dstTokenInfo.Decimals)
	if !dstClient.WithdrawalAmountValid(withdrawalAmount) {
		return nil, ErrInvalidDepositedAmount
	}

	deposit := depositData.ToNewDeposit(withdrawalAmount, dstTokenInfo.Address, dstTokenInfo.IsWrapped)

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
