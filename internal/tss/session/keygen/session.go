package keygen

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	tss2 "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ p2p.TssSession = &Session{}

type Session struct {
	sessionId string
	params    session.Params
	wg        *sync.WaitGroup

	connectedPartiesCount func() int
	partiesCount          int

	keygenParty tss.KeyGenParty

	result *tss.LocalPartyData
	err    error

	logger *logan.Entry
}

func NewSession(
	self tss.LocalKeygenParty,
	parties []p2p.Party,
	params session.Params,
	connectedPartiesCountFunc func() int,
	logger *logan.Entry,
	protocolID int,
	group curve.Curve,
) *Session {

	sessionId := session.GetKeygenSessionIdentifier(params.Id)
	switch protocolID {
	case tss.ProtocolID_ECDSA:
		return &Session{
			sessionId:             sessionId,
			params:                params,
			wg:                    &sync.WaitGroup{},
			connectedPartiesCount: connectedPartiesCountFunc,
			partiesCount:          len(parties),
			keygenParty:           tss2.SelectKeyGenByProtocol(tss.ProtocolID_ECDSA, self, parties, params.Threshold, sessionId, group, logger.WithField("component", "keygen_party")),
			logger:                logger,
		}

	case tss.ProtocolID_FROST:
		return &Session{
			sessionId:             sessionId,
			params:                params,
			wg:                    &sync.WaitGroup{},
			connectedPartiesCount: connectedPartiesCountFunc,
			partiesCount:          len(parties),
			keygenParty:           tss2.SelectKeyGenByProtocol(tss.ProtocolID_FROST, self, parties, params.Threshold, sessionId, group, logger.WithField("component", "keygen_party")),
			logger:                logger,
		}

	default:
		return &Session{}
	}
}

func (s *Session) Run(ctx context.Context) error {
	runDelay := time.Until(s.params.StartTime)
	if runDelay <= 0 {
		return errors.New("target time is in the past")
	}

	s.logger.Info(fmt.Sprintf("keygen session will start in %s", runDelay))

	select {
	case <-ctx.Done():
		s.logger.Info("keygen session cancelled")
		return nil
	case <-time.After(runDelay):
		if s.connectedPartiesCount() != s.partiesCount {
			return errors.New("cannot start keygen session: not all parties connected")
		}
	}

	s.logger.Info("keygen session started")

	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

func (s *Session) run(ctx context.Context) {
	defer s.wg.Done()

	boundedCtx, cancel := context.WithTimeout(ctx, session.BoundaryKeygenSession)
	defer cancel()

	s.keygenParty.Run(boundedCtx)
	s.result = s.keygenParty.WaitFor()
	s.logger.Info("keygen session finished")
	if s.result != nil {
		return
	}

	if err := boundedCtx.Err(); err != nil {
		s.err = err
		return
	}

	s.err = errors.New("keygen session error occurred")
}

func (s *Session) WaitFor() (*tss.LocalPartyData, error) {
	s.wg.Wait()
	return s.result, s.err
}

func (s *Session) Id() string {
	return s.sessionId
}

func (s *Session) Receive(request *p2p.SubmitRequest) error {
	if request == nil || request.Data == nil {
		return errors.New("nil request")
	}
	if request.Type != p2p.RequestType_RT_KEYGEN {
		return errors.New("invalid request type")
	}
	if request.SessionId != s.sessionId {
		return errors.New(fmt.Sprintf("session id mismatch: expected '%s', got '%s'", s.sessionId, request.SessionId))
	}

	data := &p2p.TssData{}
	if err := request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal TSS request data")
	}

	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}

	// TODO: add better error handling?
	s.keygenParty.Receive(sender, data)

	return nil
}

// RegisterIdChangeListener is a no-op for Session
func (s *Session) RegisterIdChangeListener(func(oldId, newId string)) {}

// SigningSessionInfo is a no-op for Session
func (s *Session) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
