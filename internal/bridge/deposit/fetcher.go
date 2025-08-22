package deposit

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	bridgetypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
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
	fmt.Println("fetching deposit for identifier:", identifier)
	sourceClient, err := p.clients.Client(identifier.ChainId)
	if err != nil {
		fmt.Println("error getting source clients:", err)
		return nil, errors.Wrap(err, "error getting source clients")
	}

	fmt.Println("checking transaction hash validity for:", identifier.TxHash)
	if !sourceClient.TransactionHashValid(identifier.TxHash) {
		fmt.Println("invalid transaction hash:", identifier.TxHash)
		return nil, errors.New("invalid transaction hash")
	}

	fmt.Println("getting deposit data for identifier:", identifier)
	depositData, err := sourceClient.GetDepositData(identifier)
	if err != nil {
		fmt.Println("failed to get deposit data:", err)
		return nil, errors.Wrap(err, "failed to get deposit data")
	}

	fmt.Println("deposit data fetched successfully:", depositData)
	dstClient, err := p.clients.Client(depositData.DestinationChainId)
	if err != nil {
		fmt.Println("error getting destination clients:", err)
		return nil, errors.Wrap(err, "error getting destination clients")
	}

	fmt.Println("checking destination address validity for:", depositData.DestinationAddress)
	if !dstClient.AddressValid(depositData.DestinationAddress) {
		fmt.Println("invalid destination address:", depositData.DestinationAddress)
		return nil, errors.Wrap(chain.ErrInvalidReceiverAddress, depositData.DestinationAddress)
	}

	fmt.Println("getting source token info for chain ID:", identifier.ChainId, "and token address:", depositData.TokenAddress)
	srcTokenInfo, err := p.core.GetTokenInfo(identifier.ChainId, depositData.TokenAddress)
	if err != nil {
		fmt.Println("failed to get source token info:", err)
		return nil, errors.Wrap(err, "failed to get source token info")
	}

	fmt.Println("source token info fetched successfully:", srcTokenInfo)
	token, err := p.core.GetToken(srcTokenInfo.TokenId)
	if err != nil {
		fmt.Println("failed to get source token:", err)
		return nil, errors.Wrap(err, "failed to get source token")
	}

	fmt.Println("getting destination token info for token:", token, "and destination chain ID:", depositData.DestinationChainId)
	dstTokenInfo, err := getDstTokenInfo(token, depositData.DestinationChainId)
	if err != nil {
		fmt.Println("failed to get destination token info:", err)
		return nil, errors.Wrap(err, "failed to get dst token info")
	}

	fmt.Println("destination token info fetched successfully:", dstTokenInfo)
	withdrawalAmount := transformAmount(depositData.DepositAmount, srcTokenInfo.Decimals, dstTokenInfo.Decimals)
	if !dstClient.WithdrawalAmountValid(withdrawalAmount) {
		fmt.Println("invalid withdrawal amount:", withdrawalAmount)
		return nil, chain.ErrInvalidDepositedAmount
	}

	fmt.Println("getting commission amount for withdrawal amount:", withdrawalAmount, "and commission rate:", token.CommissionRate)
	commissionAmount, err := getCommissionAmount(withdrawalAmount, token.CommissionRate)
	if err != nil {
		fmt.Println("failed to get commission amount:", err)
		return nil, errors.Wrap(err, "failed to get commission amount")
	}

	fmt.Println("commission amount calculated successfully:", commissionAmount)
	finalWithdrawalAmount := big.NewInt(0).Sub(withdrawalAmount, commissionAmount)
	if !dstClient.WithdrawalAmountValid(finalWithdrawalAmount) {
		fmt.Println("invalid final withdrawal amount:", finalWithdrawalAmount)
		return nil, errors.Wrap(chain.ErrInvalidDepositedAmount, "invalid final withdrawal amount")
	}

	fmt.Println("creating new deposit with final withdrawal amount:", finalWithdrawalAmount, "and commission amount:", commissionAmount)
	deposit := depositData.ToNewDeposit(finalWithdrawalAmount, commissionAmount,
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

// getCommissionAmount returns a commission amount basing on provided withdrawal amount and token commission rate.
func getCommissionAmount(withdrawalAmount *big.Int, commissionRate string) (*big.Int, error) {
	rate, err := sdkmath.LegacyNewDecFromStr(commissionRate)

	if err != nil {
		return big.NewInt(0), errors.Wrap(err, "failed to parse commission rate")
	}

	return rate.Mul(sdkmath.LegacyNewDecFromBigInt(withdrawalAmount)).TruncateInt().BigInt(), nil
}

func getDstTokenInfo(token bridgetypes.Token, dstChainId string) (bridgetypes.TokenInfo, error) {
	fmt.Println("getting destination token info for chain ID:", dstChainId, "in token:", token)
	for _, info := range token.Info {
		if info.ChainId == dstChainId {
			return info, nil
		}
	}

	return bridgetypes.TokenInfo{}, core.ErrDestinationTokenInfoNotFound
}
