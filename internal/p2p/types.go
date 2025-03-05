package p2p

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUnsupportedChain = status.Error(codes.InvalidArgument, "unsupported chain")
	ErrInternal         = status.Error(codes.Internal, "internal error")
)
