package common

import (
	"net/http"
	"strconv"

	apiTypes "github.com/Bridgeless-Project/tss-svc/internal/api/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/go-chi/chi/v5"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	ParamChainId = "chain_id"
	ParamTxHash  = "tx_hash"
	ParamTxNonce = "tx_nonce"
)

func ValidateIdentifier(identifier *types.DepositIdentifier) error {
	if identifier == nil {
		return errors.New("identifier is required")
	}

	return validation.Errors{
		"tx_hash":  validation.Validate(identifier.TxHash, validation.Required),
		"chain_id": validation.Validate(identifier.ChainId, validation.Required),
		"tx_nonce": validation.Validate(identifier.TxNonce, validation.Required, validation.Min(0)),
	}.Filter()
}

func ValidateChainIdentifier(identifier *types.DepositIdentifier, client chain.Client) error {
	if !client.TransactionHashValid(identifier.TxHash) {
		return errors.New("invalid transaction hash")
	}

	if client.Type() == chain.TypeZano && identifier.TxNonce != 0 {
		return errors.New("event index must be 0 for Zano chain")
	}

	return nil
}

func IdentifierFromParams(r *http.Request) (*database.DepositIdentifier, error) {
	data := &types.DepositIdentifier{
		ChainId: chi.URLParam(r, ParamChainId),
		TxHash:  chi.URLParam(r, ParamTxHash),
	}
	data.TxNonce, _ = strconv.ParseInt(chi.URLParam(r, ParamTxNonce), 10, 64)

	if err := ValidateIdentifier(data); err != nil {
		return nil, errors.Wrap(err, "invalid data")
	}

	identifier := ToDbIdentifier(data)

	return &identifier, nil
}

func ToStatusResponse(d *database.Deposit) *apiTypes.CheckWithdrawalResponse {
	result := &apiTypes.CheckWithdrawalResponse{
		DepositIdentifier: &types.DepositIdentifier{
			TxHash:  d.TxHash,
			TxNonce: d.TxNonce,
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
		CommissionAmount: d.CommissionAmount,
		DepositAsset:     d.DepositToken,
		WithdrawalAsset:  d.WithdrawalToken,
		IsWrappedAsset:   d.IsWrappedToken,
		DepositBlock:     d.DepositBlock,
		Signature:        d.Signature,
	}
	result.WithdrawalIdentifier = &types.WithdrawalIdentifier{
		TxHash:  d.WithdrawalTxHash,
		ChainId: d.WithdrawalChainId,
	}

	return result
}

func ToDbIdentifier(identifier *types.DepositIdentifier) database.DepositIdentifier {
	return database.DepositIdentifier{
		TxHash:  identifier.TxHash,
		TxNonce: identifier.TxNonce,
		ChainId: identifier.ChainId,
	}
}

func FromDbIdentifier(identifier database.DepositIdentifier) *types.DepositIdentifier {
	return &types.DepositIdentifier{
		TxHash:  identifier.TxHash,
		TxNonce: identifier.TxNonce,
		ChainId: identifier.ChainId,
	}
}

func ProtoJsonMustMarshal(msg proto.Message) []byte {
	raw, _ := protojson.Marshal(msg)
	return raw
}
