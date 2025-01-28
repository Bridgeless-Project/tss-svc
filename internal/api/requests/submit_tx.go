package requests

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	types "github.com/hyle-team/tss-svc/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func SubmitTx(ctxt context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	var (
		clients = ctx.Clients(ctxt)
		db      = ctx.DB(ctxt)
		logger  = ctx.Logger(ctxt)
		pr      = ctx.Processor(ctxt)
	)

	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}
	err := validateIdentifier(identifier)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	chain, err := clients.Client(identifier.ChainId)
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.InvalidArgument, "chain not found")
	}

	id := common.FormDepositIdentifier(identifier, chain.Type())

	tx, err := db.Get(id)
	if err != nil {
		logger.WithError(err).Error("failed to get withdrawal data from db")
		return nil, apiTypes.ErrInternal
	}
	if tx != nil {
		return nil, apiTypes.ErrTxAlreadySubmitted
	}

	deposit, err := pr.FetchDeposit(id)
	//perform saving to db
	insertErr := db.Transaction(func() error {
		if deposit == nil {
			logger.Error("got nil withdrawal after fetching data")
			return apiTypes.ErrInternal
		}
		_, insertErr := db.Insert(*deposit)
		if insertErr != nil {
			logger.WithError(insertErr).Error("failed to insert withdrawal data")
			return apiTypes.ErrInternal
		}
		return nil
	})
	if insertErr != nil {
		logger.WithError(err).Error("failed to insert withdrawal data")
		return nil, apiTypes.ErrInternal
	}
	if err != nil {
		logger.WithError(err).Error("failed to fetch withdrawal data")
		return nil, apiTypes.ErrInternal
	}

	return nil, nil
}

func validateIdentifier(identifier *types.DepositIdentifier) error {
	return validation.Errors{
		"tx_hash":  validation.Validate(identifier.TxHash, validation.Required),
		"chain_id": validation.Validate(identifier.ChainId, validation.Required),
		"tx_nonce": validation.Validate(identifier.TxNonce, validation.Min(0)),
	}.Filter()
}
