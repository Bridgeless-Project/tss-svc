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
	if err := common.ValidateIdentifier(identifier); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		data        = ctx.DB(ctxt)
		logger      = ctx.Logger(ctxt)
		clientsRepo = ctx.Clients(ctxt)
	)

	client, err := clientsRepo.Client(identifier.ChainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "unsupported chain")
	}
	if err = common.ValidateChainIdentifier(identifier, client); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

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
