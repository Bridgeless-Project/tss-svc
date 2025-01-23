package ctx

import (
	"context"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"gitlab.com/distributed_lab/logan/v3"
)

type ctxKey int

const (
	dbKey     ctxKey = iota
	loggerKey ctxKey = iota
	chainsKey ctxKey = iota
)

func DBProvider(q db.DepositsQ) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, dbKey, q)
	}
}

// DB always returns unique connection
func DB(ctx context.Context) db.DepositsQ {
	return ctx.Value(dbKey).(db.DepositsQ).New()
}

func LoggerProvider(l *logan.Entry) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {

		return context.WithValue(ctx, loggerKey, l)
	}
}
func Logger(ctx context.Context) *logan.Entry {
	return ctx.Value(loggerKey).(*logan.Entry)
}

func ChainsProvider(cm apiTypes.ChainsMap) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, chainsKey, cm)
	}
}

func Chains(ctx context.Context) apiTypes.ChainsMap {
	return ctx.Value(chainsKey).(apiTypes.ChainsMap)
}
