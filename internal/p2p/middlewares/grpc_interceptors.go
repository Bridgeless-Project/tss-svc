package middlewares

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoveryInterceptor(entry *logan.Entry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res interface{}, err error) {
		defer func() {
			if rvr := recover(); rvr != nil {
				rerr := errors.FromPanic(rvr)
				entry.WithError(rerr).WithField("method", info.FullMethod).Error("handler panicked")

				res, err = nil, status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}
