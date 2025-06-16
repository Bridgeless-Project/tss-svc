package grpc

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/api/common"
	"github.com/Bridgeless-Project/tss-svc/internal/api/ctx"
	apiTypes "github.com/Bridgeless-Project/tss-svc/internal/api/types"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (Implementation) CheckWithdrawal(ctxt context.Context, identifier *types.DepositIdentifier) (*apiTypes.CheckWithdrawalResponse, error) {
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	var (
		data   = ctx.DB(ctxt)
		logger = ctx.Logger(ctxt)
	)

	deposit, err := data.Get(common.ToDbIdentifier(identifier))
	if err != nil {
		logger.WithError(err).Error("failed to get withdrawal")
		return nil, ErrInternal
	}
	if deposit == nil {
		return nil, status.Error(codes.NotFound, "withdrawal not found")
	}

	return common.ToStatusResponse(deposit), nil
}
