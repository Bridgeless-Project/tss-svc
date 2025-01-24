package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hyle-team/tss-svc/docs"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	"github.com/hyle-team/tss-svc/internal/api/middlewares"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	"github.com/hyle-team/tss-svc/internal/bridge/deposit"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"gitlab.com/distributed_lab/logan/v3"

	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/ignite/cli/ignite/pkg/openapiconsole"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/ape"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ apiTypes.APIServer = grpcImplementation{}

type grpcImplementation struct{}

type server struct {
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
	clients bridgeTypes.ClientsRepository,
	processor *deposit.Processor,
) apiTypes.Server {
	return &server{
		grpc:   grpc,
		http:   http,
		logger: logger,

		ctxExtenders: []func(context.Context) context.Context{
			ctx.LoggerProvider(logger),
			ctx.DBProvider(db),
			ctx.ClientsProvider(clients),
			ctx.ProcessorProvider(processor),
		},
	}
}

func (s *server) RunGRPC(ctx context.Context) error {
	srv := s.grpcServer()

	// graceful shutdown
	go func() { <-ctx.Done(); srv.GracefulStop(); s.logger.Info("grpc serving stopped: context canceled") }()

	s.logger.Info("grpc serving started")
	return srv.Serve(s.grpc)
}

func (s *server) RunHTTP(ctxt context.Context) error {
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

func (s *server) httpRouter(ctxt context.Context) http.Handler {
	router := chi.NewRouter()
	router.Use(ape.LoganMiddleware(s.logger), ape.RecoverMiddleware(s.logger))

	// pointing to grpc implementation
	grpcGatewayRouter := runtime.NewServeMux()
	_ = apiTypes.RegisterAPIHandlerServer(ctxt, grpcGatewayRouter, grpcImplementation{})

	// grpc interceptor not working here
	router.With(ape.CtxMiddleware(s.ctxExtenders...)).Mount("/", grpcGatewayRouter)
	router.With(
		ape.CtxMiddleware(s.ctxExtenders...),
		// extending with websocket middleware
		middlewares.HijackedConnectionCloser(ctxt),
	).Get("/ws/check/{chain_id}/{tx_hash}/{tx_nonce}", CheckWithdrawalWs)
	//TODO: Check for ws implementation
	router.Mount("/static/api_server.swagger.json", http.FileServer(http.FS(docs.Docs)))
	router.HandleFunc("/api", openapiconsole.Handler("Signer service", "/static/api_server.swagger.json"))

	return router
}

func (s *server) grpcServer() *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			ContextExtenderInterceptor(s.ctxExtenders...),
			LoggerInterceptor(s.logger),
			// RecoveryInterceptor should be the last one
			RecoveryInterceptor(s.logger),
		),
	)

	apiTypes.RegisterAPIServer(srv, grpcImplementation{})
	reflection.Register(srv)

	return srv
}
