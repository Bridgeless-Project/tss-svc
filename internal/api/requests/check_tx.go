package requests

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/api/common"
	ctxt "github.com/hyle-team/tss-svc/internal/api/ctx"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CheckTx(ctx context.Context, identifier *types.DepositIdentifier) (*database.Deposit, error) {
	var (
		db      = ctxt.DB(ctx)
		logger  = ctxt.Logger(ctx)
		clients = ctxt.Clients(ctx)
	)
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	client, err := clients.Client(identifier.ChainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chain id")
	}
	err = validateIdentifier(identifier)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id := common.FormDepositIdentifier(identifier, client.Type())

	tx, err := db.Get(id)
	if err != nil {
		logger.WithError(err).Error("failed to get deposit")
		return nil, apiTypes.ErrInternal
	}
	if tx == nil {
		return nil, status.Error(codes.NotFound, "deposit not found")
	}

	return tx, nil
}
