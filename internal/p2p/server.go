package p2p

import (
	"context"
	"net"
	"sync"

	"github.com/hyle-team/tss-svc/internal/p2p/middlewares"
	"gitlab.com/distributed_lab/logan/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ P2PServer = &Server{}

type Server struct {
	status   PartyStatus
	statusM  sync.RWMutex
	manager  *SessionManager
	listener net.Listener
	logger   *logan.Entry
}

func NewServer(listener net.Listener,
	manager *SessionManager,
	logger *logan.Entry,
) *Server {
	return &Server{
		status:   PartyStatus_PS_UNKNOWN,
		manager:  manager,
		listener: listener,
		logger:   logger,
	}
}

func (s *Server) SetStatus(status PartyStatus) {
	s.statusM.Lock()
	defer s.statusM.Unlock()

	s.status = status
}

func (s *Server) Run(ctx context.Context) error {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// RecoveryInterceptor should be the last one
			middlewares.RecoveryInterceptor(s.logger),
		))
	RegisterP2PServer(srv, s)
	reflection.Register(srv)

	// graceful shutdown
	go func() { <-ctx.Done(); srv.GracefulStop() }()

	return srv.Serve(s.listener)
}

func (s *Server) Status(ctx context.Context, empty *emptypb.Empty) (*StatusResponse, error) {
	s.statusM.RLock()
	defer s.statusM.RUnlock()

	return &StatusResponse{Status: s.status}, nil
}

func (s *Server) Submit(ctx context.Context, request *SubmitRequest) (*emptypb.Empty, error) {
	// TODO: auth check
	if err := s.manager.Receive(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetSigningSessionInfo(ctxt context.Context, request *SigningSessionInfoRequest) (*SigningSessionInfo, error) {
	s.statusM.RLock()
	st := s.status
	s.statusM.RUnlock()

	if st != PartyStatus_PS_SIGN {
		return nil, status.Error(codes.FailedPrecondition, "party is not in signing state")
	}

	session, err := s.manager.GetSigningSession(request.ChainId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return session.SigningSessionInfo(), nil
}
