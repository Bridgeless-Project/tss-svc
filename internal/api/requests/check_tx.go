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
		db     = ctxt.DB(ctx)
		logger = ctxt.Logger(ctx)
		chains = ctxt.Chains(ctx)
	)
	if identifier == nil {
		logger.Error("empty identifier")

		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	chain, ok := chains[identifier.ChainId]
	if !ok {

		return nil, status.Error(codes.NotFound, "chain not found")
	}
	id := common.FormDepositIdentifier(identifier, chain.Type)

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
