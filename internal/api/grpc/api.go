package grpc

import (
	"github.com/hyle-team/tss-svc/internal/api/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal           = status.Error(codes.Internal, "internal error")
	ErrTxAlreadySubmitted = status.Error(codes.AlreadyExists, "transaction already submitted")
	ErrDepositPending     = status.Error(codes.FailedPrecondition, "deposit pending")
)

var _ types.APIServer = Implementation{}

type Implementation struct{}
