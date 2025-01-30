package requests

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	"github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func SubmitTx(ctxt context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	var (
		clientsRepo = ctx.Clients(ctxt)
		data        = ctx.DB(ctxt)
		logger      = ctx.Logger(ctxt)
		processor   = ctx.Processor(ctxt)
	)

	client, err := clientsRepo.Client(identifier.ChainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chains id")
	}
	if err = validateIdentifier(identifier, client); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tx, err := data.Get(common.FormDepositIdentifier(identifier, client.Type()))
	if err != nil {
		logger.WithError(err).Error("failed to get withdrawal data from db")
		return nil, apiTypes.ErrInternal
	}
	if tx != nil {
		return nil, apiTypes.ErrTxAlreadySubmitted
	}

	id := db.DepositIdentifier{
		ChainId: identifier.ChainId,
		TxHash:  identifier.TxHash,
		TxNonce: int(identifier.TxNonce),
	}

	deposit, err := processor.FetchDeposit(id)
	if err != nil {
		if clients.IsPendingDepositError(err) {
			return nil, apiTypes.ErrDepositPending
		}
		if clients.IsInvalidDepositError(err) {
			// TODO: insert in db
			return nil, status.Error(codes.InvalidArgument, "invalid deposit")
		}

		logger.WithError(err).Error("failed to fetch deposit")
		return nil, apiTypes.ErrInternal
	}

	if _, err = data.Insert(*deposit); err != nil {
		logger.WithError(err).Error("failed to save deposit")
		return nil, apiTypes.ErrInternal
	}

	return nil, nil
}

func validateIdentifier(identifier *types.DepositIdentifier, client clients.Client) error {
	err := validation.Errors{
		"tx_hash":  validation.Validate(identifier.TxHash, validation.Required),
		"chain_id": validation.Validate(identifier.ChainId, validation.Required),
		"tx_nonce": validation.Validate(identifier.TxNonce, validation.Min(0)),
	}.Filter()
	if err != nil {
		return err
	}

	if !client.TransactionHashValid(identifier.TxHash) {
		return errors.New("invalid transaction hash")
	}

	return nil
}
