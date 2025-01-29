package middlewares

import (
	"context"
	"time"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ContextExtenderInterceptor(extenders ...func(context.Context) context.Context) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		for _, extend := range extenders {
			ctx = extend(ctx)
		}

		return handler(ctx, req)
	}
}

func LoggerInterceptor(entry *logan.Entry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := entry.WithField("method", info.FullMethod)
		start := time.Now()

		logger.Info("request started")

		res, err := handler(ctx, req)

		logger.WithFields(logan.F{
			"duration": time.Since(start),
			"status":   status.Code(err),
		}).Info("request finished")

		return res, err
	}
}

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
