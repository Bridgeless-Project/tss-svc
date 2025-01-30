package types

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal           = status.Error(codes.Internal, "internal error")
	ErrTxAlreadySubmitted = status.Error(codes.AlreadyExists, "transaction already submitted")
	ErrDepositPending     = status.Error(codes.FailedPrecondition, "deposit pending")

	ErrInvalidTxNonce = errors.New("invalid origin tx nonce")
	ErrInvalidTxHash  = errors.New("invalid origin tx hash")
	ErrInvalidChainId = errors.New("invalid chains id")
)

type Server interface {
	RunGRPC(ctx context.Context) error
	RunHTTP(ctx context.Context) error
}
