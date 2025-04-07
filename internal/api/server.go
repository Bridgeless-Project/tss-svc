package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/deposit"
	coreConnector "github.com/hyle-team/tss-svc/internal/core/connector"
	"github.com/hyle-team/tss-svc/internal/p2p/broadcast"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hyle-team/tss-svc/api"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	srvgrpc "github.com/hyle-team/tss-svc/internal/api/grpc"
	srvhttp "github.com/hyle-team/tss-svc/internal/api/http"
	"github.com/hyle-team/tss-svc/internal/api/middlewares"
	"github.com/hyle-team/tss-svc/internal/api/types"
	"github.com/hyle-team/tss-svc/internal/core"
	"gitlab.com/distributed_lab/logan/v3"

	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/ignite/cli/ignite/pkg/openapiconsole"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/ape"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpc net.Listener
	http net.Listener

	logger       *logan.Entry
	ctxExtenders []func(context.Context) context.Context
}

// NewServer creates a new GRPC server.
func NewServer(
	grpc net.Listener,
	http net.Listener,
	db db.DepositsQ,
	logger *logan.Entry,
	clients chain.Repository,
	processor *deposit.Fetcher,
	broadcaster *broadcast.Broadcaster,
	self core.Address,
	connector *coreConnector.Connector,
) *Server {
	return &Server{
		grpc:   grpc,
		http:   http,
		logger: logger,

		ctxExtenders: []func(context.Context) context.Context{
			ctx.LoggerProvider(logger),
			ctx.DBProvider(db),
			ctx.ClientsProvider(clients),
			ctx.FetcherProvider(processor),
			ctx.BroadcasterProvider(broadcaster),
			ctx.SelfProvider(self),
			ctx.CoreConnectorProvider(connector),
		},
	}
}

func (s *Server) RunGRPC(ctx context.Context) error {
	srv := s.grpcServer()

	// graceful shutdown
	go func() { <-ctx.Done(); srv.GracefulStop(); s.logger.Info("grpc serving stopped: context canceled") }()

	s.logger.Info("grpc serving started")
	return srv.Serve(s.grpc)
}

func (s *Server) RunHTTP(ctxt context.Context) error {
	srv := &http.Server{Handler: s.httpRouter(ctxt)}

	// graceful shutdown
	go func() {
		<-ctxt.Done()
		shutdownDeadline, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownDeadline); err != nil {
			s.logger.WithError(err).Error("failed to shutdown http server")
		}
		s.logger.Info("http serving stopped: context canceled")
	}()

	s.logger.Info("http serving started")
	if err := srv.Serve(s.http); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) httpRouter(ctxt context.Context) http.Handler {
	router := chi.NewRouter()
	router.Use(
		ape.LoganMiddleware(s.logger),
		ape.RecoverMiddleware(s.logger),
		ape.CtxMiddleware(s.ctxExtenders...),
	)

	// pointing to grpc implementation
	grpcGatewayRouter := runtime.NewServeMux()
	_ = types.RegisterAPIHandlerServer(ctxt, grpcGatewayRouter, srvgrpc.Implementation{})

	router.Mount("/", grpcGatewayRouter)
	router.With(middlewares.HijackedConnectionCloser(ctxt)).Get("/ws/check/{chain_id}/{tx_hash}/{tx_nonce}", srvhttp.CheckWithdrawalWs)
	router.Mount("/static/api_server.swagger.json", http.FileServer(http.FS(api.Docs)))
	router.HandleFunc("/api", openapiconsole.Handler("TSS service API", "/static/api_server.swagger.json"))

	return router
}

func (s *Server) grpcServer() *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middlewares.ContextExtenderInterceptor(s.ctxExtenders...),
			middlewares.LoggerInterceptor(s.logger),
			// RecoveryInterceptor should be the last one
			middlewares.RecoveryInterceptor(s.logger),
		),
	)

	types.RegisterAPIServer(srv, srvgrpc.Implementation{})
	reflection.Register(srv)

	return srv
}
