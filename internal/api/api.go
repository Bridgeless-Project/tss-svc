package api

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/requests"
	types2 "github.com/hyle-team/tss-svc/internal/api/types"
	"github.com/hyle-team/tss-svc/internal/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ types2.APIServer = grpcImplementation{}

func (grpcImplementation) SubmitWithdrawal(ctx context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {

	return requests.SubmitTx(ctx, identifier)
}

func (grpcImplementation) CheckWithdrawal(ctx context.Context, identifier *types.DepositIdentifier) (*types2.CheckWithdrawalResponse, error) {
	tx, err := requests.CheckTx(ctx, identifier)
	if err != nil {
		return nil, err
	}

	return common.ToStatusResponse(tx), nil
}
