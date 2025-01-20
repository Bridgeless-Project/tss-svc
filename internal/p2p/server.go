package p2p

import (
	"context"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ P2PServer = &Server{}

type Server struct {
	status  PartyStatus
	statusM sync.RWMutex

	manager *SessionManager

	listener net.Listener
}

func NewServer(listener net.Listener, manager *SessionManager) *Server {
	return &Server{
		status:   PartyStatus_UNKNOWN,
		manager:  manager,
		listener: listener,
	}
}

func (s *Server) SetStatus(status PartyStatus) {
	s.statusM.Lock()
	defer s.statusM.Unlock()

	s.status = status
}

func (s *Server) Run(ctx context.Context) error {
	// TODO: add interceptors (log, recovery etc)
	srv := grpc.NewServer()
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
