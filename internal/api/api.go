package api

import (
	"context"

	"github.com/hyle-team/tss-svc/internal/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ APIServer = &server{}

type server struct {
}

func (s *server) SubmitWithdrawal(ctx context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (s *server) CheckWithdrawal(ctx context.Context, identifier *types.DepositIdentifier) (*CheckWithdrawalResponse, error) {
	//TODO implement me
	panic("implement me")
}
