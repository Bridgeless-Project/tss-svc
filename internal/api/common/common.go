package common

import (
	apiTyoes "github.com/hyle-team/tss-svc/internal/api/types"
	chainTypes "github.com/hyle-team/tss-svc/internal/bridge/chain"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
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
