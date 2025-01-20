package ctx

import (
	"context"
	"github.com/hyle-team/tss-svc/internal/db"
)

type ctxKey int

const (
	dbKey ctxKey = iota
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
