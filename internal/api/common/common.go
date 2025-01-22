package common

import (
	apiTyoes "github.com/hyle-team/tss-svc/internal/api/types"
	chainTypes "github.com/hyle-team/tss-svc/internal/bridge/chain"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"math/big"
	"strconv"
)

func ToStatusResponse(d *database.Deposit) *apiTyoes.CheckWithdrawalResponse {
	result := &apiTyoes.CheckWithdrawalResponse{
		DepositIdentifier: &types.DepositIdentifier{
			TxHash:  d.TxHash,
			TxNonce: int64(d.TxNonce),
			ChainId: d.ChainId,
		},
		TransferData: &types.TransferData{
			Sender:           d.Depositor,
			Receiver:         *d.Receiver,
			DepositAmount:    *d.DepositAmount,
			WithdrawalAmount: *d.WithdrawalAmount,
			DepositAsset:     *d.DepositToken,
			WithdrawalAsset:  *d.WithdrawalToken,
			IsWrappedAsset:   strconv.FormatBool(*d.IsWrappedToken),
			DepositBlock:     *d.DepositBlock,
			Signature:        d.Signature,
		},
		WithdrawalStatus: d.WithdrawalStatus,
	}
	if d.WithdrawalTxHash != nil && d.WithdrawalChainId != nil {
		result.WithdrawalIdentifier = &types.WithdrawalIdentifier{
			TxHash:  *d.WithdrawalTxHash,
			ChainId: *d.WithdrawalChainId,
		}
	}

	return result
}

func FormDepositIdentifier(identifier *types.DepositIdentifier, chainType chainTypes.Type) database.DepositIdentifier {
	if chainType == chainTypes.TypeZano {
		return database.DepositIdentifier{
			TxHash:  identifier.TxHash,
			ChainId: identifier.ChainId,
		}
	}

	return database.DepositIdentifier{
		TxHash:  identifier.TxHash,
		TxNonce: int(identifier.TxNonce),
		ChainId: identifier.ChainId,
	}
}

func CheckIfDepositExists(identifier database.DepositIdentifier, db database.DepositsQ) (bool, error) {
	deposit, err := db.Get(identifier)
	if err != nil {
		return false, err
	}

	return deposit != nil, nil
}

func GetDepositData(identifier database.DepositIdentifier, p chainTypes.Chain, db database.DepositsQ) error {
	withdrawalChainId := "dscds"
	depositor := "dsfdsf"
	receiver := "depositToken"
	depositToken := "sd"
	withdrawalToken := "sd"
	signature := "dddddd"
	isWrappedToken := false
	depositBlock := int64(0)
	depositAmount := big.NewInt(1323213).String()
	withdrwalAmount := big.NewInt(1).String()

	deposit := &database.Deposit{
		DepositIdentifier: identifier,
		WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING,
		WithdrawalChainId: &withdrawalChainId,
		Depositor:         &depositor,
		Receiver:          &receiver,
		DepositAmount:     &depositAmount,
		WithdrawalAmount:  &withdrwalAmount,
		DepositToken:      &depositToken,
		WithdrawalToken:   &withdrawalToken,
		IsWrappedToken:    &isWrappedToken,
		Signature:         &signature,
		DepositBlock:      &depositBlock,
	}

	// TODO: get deposit data from network (if data is invalid pass status INVALID)
	// TODO: pass deposit data to db

	_, err := db.Insert(*deposit)
	if err != nil {
		return err
	}

	return nil
}
