package p2p

import (
	"context"
	"net"
	"sync"

	"github.com/hyle-team/tss-svc/internal/tss/session"
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

	manager session.Manager

	listener net.Listener
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
	if err := s.manager.Receive(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &emptypb.Empty{}, nil
}
