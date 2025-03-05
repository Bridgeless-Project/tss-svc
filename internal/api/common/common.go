package common

import (
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	"github.com/hyle-team/tss-svc/internal/bridge/chains"
	"github.com/hyle-team/tss-svc/internal/bridge/clients"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ValidateIdentifier(identifier *types.DepositIdentifier, client clients.Client) error {
	err := validation.Errors{
		"tx_hash":  validation.Validate(identifier.TxHash, validation.Required),
		"chain_id": validation.Validate(identifier.ChainId, validation.Required),
	}.Filter()
	if err != nil {
		return err
	}

	if !client.TransactionHashValid(identifier.TxHash) {
		return errors.New("invalid transaction hash")
	}

	// If chain type is Zano event index always is 0
	if client.Type() == chains.TypeZano {
		identifier.TxNonce = 0
	}

	return nil
}

func ToStatusResponse(d *database.Deposit) *apiTypes.CheckWithdrawalResponse {
	result := &apiTypes.CheckWithdrawalResponse{
		DepositIdentifier: &types.DepositIdentifier{
			TxHash:  d.TxHash,
			TxNonce: uint32(d.TxNonce),
			ChainId: d.ChainId,
		},
		WithdrawalStatus: d.WithdrawalStatus,
	}

	if d.WithdrawalStatus == types.WithdrawalStatus_WITHDRAWAL_STATUS_INVALID {
		return result
	}

	result.TransferData = &types.TransferData{
		Sender:           d.Depositor,
		Receiver:         d.Receiver,
		DepositAmount:    d.DepositAmount,
		WithdrawalAmount: d.WithdrawalAmount,
		DepositAsset:     d.DepositToken,
		WithdrawalAsset:  d.WithdrawalToken,
		IsWrappedAsset:   d.IsWrappedToken,
		DepositBlock:     d.DepositBlock,
		Signature:        d.Signature,
	}

	if d.WithdrawalTxHash != nil {
		result.WithdrawalIdentifier = &types.WithdrawalIdentifier{
			TxHash:  *d.WithdrawalTxHash,
			ChainId: d.WithdrawalChainId,
		}
	}

	return result
}

func ToDbIdentifier(identifier *types.DepositIdentifier) database.DepositIdentifier {
	return database.DepositIdentifier{
		TxHash:  identifier.TxHash,
		TxNonce: int(identifier.TxNonce),
		ChainId: identifier.ChainId,
	}
}

func ProtoJsonMustMarshal(msg proto.Message) []byte {
	raw, _ := protojson.Marshal(msg)
	return raw
}

func IsSupportedChain(repository clients.Repository) validation.Rule {
	return validation.By(func(value interface{}) error {
		id, ok := value.(string)
		if !ok {
			return errors.New(fmt.Sprintf("invalid chain type, expected string, got: %T", value))
		}

		if !repository.SupportsChain(id) {
			return clients.ErrChainNotSupported
		}
		return nil
	})
}
