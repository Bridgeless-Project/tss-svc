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
		chains = ctx.Chains(ctxt)
		db     = ctx.DB(ctxt)
		logger = ctx.Logger(ctxt)
	)

	if identifier == nil {
		logger.Error("empty identifier")
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}
	err := validateIdentifier(identifier)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	chain, ok := chains[identifier.ChainId]
	if !ok {
		return &emptypb.Empty{}, status.Error(codes.NotFound, "chain not found")
	}
	id := common.FormDepositIdentifier(identifier, chain.Type)
	tx, err := db.Get(id)
	if err != nil {
		logger.WithError(err).Error("failed to get deposit data from db")
		return nil, apiTypes.ErrFailedGetDepositData
	}
	if tx != nil {
		return nil, apiTypes.ErrTxAlreadySubmitted
	}

	if err := common.SaveDepositData(id, chain, db); err != nil {
		logger.WithError(err).Error("failed to save deposit data")
		return nil, apiTypes.ErrFailedSaveDepositData
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
