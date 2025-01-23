package requests

import (
	"context"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/processor"
	types "github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func SubmitTx(ctxt context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	var (
		clients   = ctx.Clients(ctxt)
		db        = ctx.DB(ctxt)
		logger    = ctx.Logger(ctxt)
		processor = ctx.Processor(ctxt)
	)

	if identifier == nil {
		logger.Error("empty identifier")
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}
	err := validation.Errors{
		"tx_hash":  validation.Validate(identifier.TxHash, validation.Required),
		"chain_id": validation.Validate(identifier.ChainId, validation.Required),
		"tx_nonce": validation.Validate(identifier.TxNonce, validation.Min(0)),
	}.Filter()

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	chain, err := clients.Client(identifier.ChainId)
	if err != nil {
		return &emptypb.Empty{}, status.Error(codes.NotFound, "chain not found")
	}
	id := common.FormDepositIdentifier(identifier, chain.Type())
	exists, err := common.CheckIfDepositExists(id, db)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "already submitted")
	}
	if err := saveDepositData(identifier, db, *processor, logger); err != nil {
		if errors.Is(err, database.ErrAlreadySubmitted) {
			return nil, status.Error(codes.AlreadyExists, "already submitted")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return nil, nil
}

func saveDepositData(identifier *types.DepositIdentifier, db database.DepositsQ, processor processor.Processor, logger *logan.Entry) error {
	deposit, err := processor.FetchDepositData(identifier, logger)
	_, insertErr := db.Insert(*deposit)
	if insertErr != nil {
		return errors.Wrap(insertErr, "failed to insert deposit data")
	}
	if err != nil {
		return err
	}

	return nil
}
