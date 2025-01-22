package types

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal           = status.Error(codes.Internal, "internal error")
	ErrTxAlreadySubmitted = status.Error(codes.AlreadyExists, "transaction already submitted")
	ErrInvalidTxNonce     = errors.New("invalid origin tx nonce")
	ErrInvalidTxHash      = errors.New("invalid origin tx hash")

	ErrInvalidChainId = errors.New("invalid chain id")
)

type Server interface {
	RunGRPC(ctx context.Context) error
	RunHTTP(ctx context.Context) error
}

type ChainsMap map[string]chain.Chain
