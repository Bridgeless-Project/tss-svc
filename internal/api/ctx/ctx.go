package ctx

import (
	"context"

	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	"github.com/hyle-team/tss-svc/internal/db"
	"gitlab.com/distributed_lab/logan/v3"
)

type ctxKey int

const (
	dbKey        ctxKey = iota
	loggerKey    ctxKey = iota
	clientsKey   ctxKey = iota
	processorKey ctxKey = iota
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

func ClientsProvider(cr bridgeTypes.ClientsRepository) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, clientsKey, cr)
	}
}

func Clients(ctx context.Context) bridgeTypes.ClientsRepository {
	return ctx.Value(clientsKey).(bridgeTypes.ClientsRepository)
}

func ProcessorProvider(processor *withdrawal.Processor) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, processorKey, processor)
	}
}

func Processor(ctx context.Context) *withdrawal.Processor {
	return ctx.Value(processorKey).(*withdrawal.Processor)
}
