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
	ErrInvalidOriginTxId  = errors.New("invalid origin tx id")
)

type Server interface {
	RunGRPC(ctx context.Context) error
	RunHTTP(ctx context.Context) error
}

type ChainsMap map[string]chain.Type
