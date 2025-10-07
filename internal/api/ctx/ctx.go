package ctx

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/api/health"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"gitlab.com/distributed_lab/logan/v3"
)

type ctxKey int

const (
	dbKey ctxKey = iota
	loggerKey
	clientsKey
	processorKey
	broadcasterKey
	selfKey
	coreConnectorKey
	healthCheckerKey
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

func ClientsProvider(cr bridgeTypes.Repository) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, clientsKey, cr)
	}
}

func Clients(ctx context.Context) bridgeTypes.Repository {
	return ctx.Value(clientsKey).(bridgeTypes.Repository)
}

func FetcherProvider(processor *deposit.Fetcher) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, processorKey, processor)
	}
}

func Fetcher(ctx context.Context) *deposit.Fetcher {
	return ctx.Value(processorKey).(*deposit.Fetcher)
}

func BroadcasterProvider(b *broadcast.Broadcaster) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, broadcasterKey, b)
	}
}

func Broadcaster(ctx context.Context) *broadcast.Broadcaster {
	return ctx.Value(broadcasterKey).(*broadcast.Broadcaster)
}

func SelfProvider(self core.Address) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, selfKey, self)
	}
}

func Self(ctx context.Context) core.Address {
	return ctx.Value(selfKey).(core.Address)
}

func CoreConnectorProvider(connector *coreConnector.Connector) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context { return context.WithValue(ctx, coreConnectorKey, connector) }
}

func CoreConnector(ctx context.Context) *coreConnector.Connector {
	return ctx.Value(coreConnectorKey).(*coreConnector.Connector)
}

func HealthCheckerProvider(checker *health.Checker) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, healthCheckerKey, checker)
	}
}

func HealthChecker(ctx context.Context) *health.Checker {
	return ctx.Value(healthCheckerKey).(*health.Checker)
}
