package p2p

import (
	"context"
	"crypto/tls"
	"net"
	"sync"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p/middlewares"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

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

	manager     *SessionManager
	authParties *AuthorizedParties

	tlsConfig *tls.Config
	listener  net.Listener

	logger *logan.Entry
}

func NewServer(
	listener net.Listener,
	manager *SessionManager,
	parties []Party,
	tlsCert tls.Certificate,
	logger *logan.Entry,
) *Server {
	clientCAs, authorizedParties, err := ConfigurePartiesCertPool(parties)
	if err != nil {
		panic(errors.Wrap(err, "failed to configure parties cert pool"))
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
		Certificates: []tls.Certificate{tlsCert},
	}

	return &Server{
		status:      PartyStatus_PS_UNKNOWN,
		manager:     manager,
		listener:    listener,
		logger:      logger,
		tlsConfig:   tlsConfig,
		authParties: authorizedParties,
	}
}

func (s *Server) SetStatus(status PartyStatus) {
	s.statusM.Lock()
	defer s.statusM.Unlock()

	s.status = status
}

func (s *Server) Run(ctx context.Context) error {
	srv := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(s.tlsConfig)),
		grpc.ChainUnaryInterceptor(
			// RecoveryInterceptor should be the last one
			middlewares.RecoveryInterceptor(s.logger),
		),
	)
	RegisterP2PServer(srv, s)
	reflection.Register(srv)

	// graceful shutdown
	go func() { <-ctx.Done(); srv.GracefulStop() }()

	return srv.Serve(s.listener)
}

func (s *Server) Status(context.Context, *emptypb.Empty) (*StatusResponse, error) {
	s.statusM.RLock()
	defer s.statusM.RUnlock()

	return &StatusResponse{Status: s.status}, nil
}

func (s *Server) Submit(ctx context.Context, request *SubmitRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	authorizedParty, err := s.authorizeParty(ctx)
	if err != nil {
		return nil, err
	}
	if authorizedParty == nil {
		return nil, status.Error(codes.PermissionDenied, "party is not in authorized parties list")
	}
	if authorizedParty.String() != request.Sender {
		return nil, status.Error(codes.PermissionDenied, "party is not authorized to send this request")
	}

	if err := s.manager.Receive(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetSigningSessionInfo(_ context.Context, request *SigningSessionInfoRequest) (*SigningSessionInfo, error) {
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

func (s *Server) authorizeParty(ctx context.Context) (*core.Address, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no tls info found")
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no client certificate found")
	}

	clientCert := tlsInfo.State.PeerCertificates[0]

	return s.authParties.Get(clientCert), nil
}
