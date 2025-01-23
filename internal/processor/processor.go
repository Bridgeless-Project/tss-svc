package processor

import (
	"github.com/hyle-team/tss-svc/internal/api/common"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"math/big"
)

type Processor struct {
	core    types.Bridger
	clients bridgeTypes.ClientsRepository
	db      database.DepositsQ
}

func NewProcessor(clients bridgeTypes.ClientsRepository, db database.DepositsQ, core types.Bridger) *Processor {
	return &Processor{
		clients: clients,
		db:      db,
		core:    core,
	}
}

func (p *Processor) FetchDepositData(identifier *types.DepositIdentifier, logger *logan.Entry) (*database.Deposit, error) {
	//get source chain client
	// error is ignored as chainId is checked before
	sourceClient, _ := p.clients.Client(identifier.ChainId)

	// form db identifier
	dbIdentifier := common.FormDepositIdentifier(identifier, sourceClient.Type())
	deposit := emptyDeposit(dbIdentifier)

	//get deposit data from network
	depositData, err := sourceClient.GetDepositData(dbIdentifier)
	if err != nil {
		return deposit, errors.Wrap(err, "deposit data not found")
	}
	logger.Debug(depositData)
	dstClient, err := p.clients.Client(depositData.DestinationChainId)
	if err != nil {
		return deposit, errors.Wrap(err, "failed to fetch deposit data")
	}

	if !dstClient.AddressValid(depositData.DestinationAddress) {
		return deposit, errors.Wrap(bridgeTypes.ErrInvalidReceiverAddress, depositData.DestinationAddress)
	}

	srcTokenInfo, err := p.core.GetTokenInfo(identifier.ChainId, depositData.TokenAddress)
	if err != nil {
		deposit.WithdrawalStatus = types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED
		return deposit, errors.Wrap(err, "failed to get source token info")
	}

	dstTokenInfo, err := p.core.GetTokenInfo(identifier.ChainId, depositData.DestinationTokenAddress)
	if err != nil {
		deposit.WithdrawalStatus = types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED
		return deposit, errors.Wrap(err, "failed to get destination token info")
	}
	depositData.WithdrawalAmount = transformAmount(depositData.DepositAmount, srcTokenInfo.Decimals, dstTokenInfo.Decimals)
	if !dstClient.WithdrawalAmountValid(depositData.WithdrawalAmount) {
		deposit.WithdrawalStatus = types.WithdrawalStatus_WITHDRAWAL_STATUS_FAILED
		return nil, bridgeTypes.ErrInvalidDepositedAmount
	}

	//set deposit data to deposit structure
	depositAmount := depositData.DepositAmount.String()
	withdrawalAmount := depositData.WithdrawalAmount.String()
	deposit = &database.Deposit{
		DepositIdentifier: dbIdentifier,
		Depositor:         &depositData.SourceAddress,
		DepositAmount:     &depositAmount,
		DepositToken:      &depositData.TokenAddress,
		Receiver:          &depositData.DestinationAddress,
		WithdrawalToken:   &dstTokenInfo.Address,
		DepositBlock:      &depositData.Block,
		WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING,
		WithdrawalChainId: &depositData.DestinationChainId,
		WithdrawalAmount:  &withdrawalAmount,
		IsWrappedToken:    &depositData.IsWrappedToken,
	}

	return deposit, nil

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

func emptyDeposit(identifier database.DepositIdentifier) *database.Deposit {
	empty := ""
	emptyInt := int64(0)
	f := false
	return &database.Deposit{
		DepositIdentifier: identifier,
		Depositor:         &empty,
		DepositAmount:     &empty,
		DepositToken:      &empty,
		Receiver:          &empty,
		WithdrawalToken:   &empty,
		DepositBlock:      &emptyInt,
		WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_INVALID,
		WithdrawalTxHash:  &empty,
		WithdrawalChainId: &empty,
		WithdrawalAmount:  &empty,
		IsWrappedToken:    &f,
		Signature:         &empty,
	}

}
