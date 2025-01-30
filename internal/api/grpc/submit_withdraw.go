package grpc

import (
	"context"

	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	"github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (Implementation) SubmitWithdrawal(ctxt context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
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
		return nil, status.Error(codes.InvalidArgument, "unsupported chain")
	}
	if err = common.ValidateIdentifier(identifier, client); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	exists, err := data.Exists(common.ToExistenceCheck(identifier, client.Type()))
	if err != nil {
		logger.WithError(err).Error("failed to check if deposit exists")
		return nil, ErrInternal
	}
	if exists {
		return nil, ErrTxAlreadySubmitted
	}

	id := db.DepositIdentifier{
		ChainId: identifier.ChainId,
		TxHash:  identifier.TxHash,
		TxNonce: int(identifier.TxNonce),
	}

	deposit, err := processor.FetchDeposit(id)
	if err != nil {
		if clients.IsPendingDepositError(err) {
			return nil, ErrDepositPending
		}
		if clients.IsInvalidDepositError(err) {
			// TODO: insert in db
			return nil, status.Error(codes.InvalidArgument, "invalid deposit")
		}

		logger.WithError(err).Error("failed to fetch deposit")
		return nil, ErrInternal
	}

	if _, err = data.Insert(*deposit); err != nil {
		logger.WithError(err).Error("failed to save deposit")
		return nil, ErrInternal
	}

	return nil, nil
}
