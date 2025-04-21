package keygen

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ p2p.TssSession = &Session{}

type Session struct {
	sessionId string
	params    session.Params
	wg        *sync.WaitGroup

	connectedPartiesCount func() int
	partiesCount          int

	keygenParty interface {
		Run(ctx context.Context)
		WaitFor() *keygen.LocalPartySaveData
		Receive(sender core.Address, data *p2p.TssData)
	}

	result *keygen.LocalPartySaveData
	err    error

	logger *logan.Entry
}

func NewSession(
	self tss.LocalKeygenParty,
	parties []p2p.Party,
	params session.Params,
	connectedPartiesCountFunc func() int,
	logger *logan.Entry,
) *Session {
	sessionId := session.GetKeygenSessionIdentifier(params.Id)
	return &Session{
		sessionId:             sessionId,
		params:                params,
		wg:                    &sync.WaitGroup{},
		connectedPartiesCount: connectedPartiesCountFunc,
		partiesCount:          len(parties),
		keygenParty:           tss.NewKeygenParty(self, parties, sessionId, logger.WithField("component", "keygen_party")),
		logger:                logger,
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
	} else {
		s.err = errors.New("keygen session error occurred")
	}
}

func (s *Session) WaitFor() (*keygen.LocalPartySaveData, error) {
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
