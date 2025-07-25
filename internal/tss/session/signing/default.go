package signing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type DefaultSessionParams struct {
	session.Params
	SigningData []byte
}

type DefaultSession struct {
	sessionId string

	params DefaultSessionParams
	logger *logan.Entry
	wg     *sync.WaitGroup

	connectedPartiesCount func() int
	partiesCount          int

	signingParty interface {
		Run(ctx context.Context)
		WaitFor() *common.SignatureData
		Receive(sender core.Address, data *p2p.TssData)
	}

	result *common.SignatureData
	err    error
}

func NewDefaultSession(
	self tss.LocalSignParty,
	params DefaultSessionParams,
	parties []p2p.Party,
	connectedPartiesCountFunc func() int,
	logger *logan.Entry,
) *DefaultSession {
	sessionId := session.GetDefaultSigningSessionIdentifier(params.Id)
	return &DefaultSession{
		sessionId:             sessionId,
		params:                params,
		wg:                    &sync.WaitGroup{},
		logger:                logger,
		connectedPartiesCount: connectedPartiesCountFunc,
		signingParty: tss.NewSignParty(self, sessionId, logger).
			WithSigningData(params.SigningData).
			WithParties(parties),
		partiesCount: len(parties),
	}
}

func (s *DefaultSession) Run(ctx context.Context) error {
	runDelay := time.Until(s.params.StartTime)
	if runDelay <= 0 {
		return errors.New("target time is in the past")
	}

	s.logger.Info(fmt.Sprintf("signing session will start in %s", runDelay))

	select {
	case <-ctx.Done():
		s.logger.Info("signing session cancelled")
		return nil
	case <-time.After(runDelay):
		if s.connectedPartiesCount() != s.partiesCount {
			return errors.New("cannot start signing session: not all parties connected")
		}
	}

	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

func (s *DefaultSession) run(ctx context.Context) {
	defer s.wg.Done()

	boundedCtx, cancel := context.WithTimeout(ctx, session.BoundarySign)
	defer cancel()

	s.signingParty.Run(boundedCtx)
	s.result = s.signingParty.WaitFor()
	if s.result != nil {
		return
	}

	if err := boundedCtx.Err(); err != nil {
		s.err = err
	} else {
		s.err = errors.New("signing session error occurred")
	}
}

func (s *DefaultSession) WaitFor() (*common.SignatureData, error) {
	s.wg.Wait()
	return s.result, s.err
}

func (s *DefaultSession) Id() string {
	return s.sessionId
}

func (s *DefaultSession) Receive(request *p2p.SubmitRequest) error {
	if request == nil || request.Data == nil {
		return errors.New("nil request")
	}
	if request.Type != p2p.RequestType_RT_SIGN {
		return errors.New("invalid request type")
	}
	if request.SessionId != s.sessionId {
		return errors.New(fmt.Sprintf("session id mismatch: expected '%s', got '%s'", s.sessionId, request.SessionId))
	}

	data := &p2p.TssData{}
	if err := request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal TSS request signingData")
	}

	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}

	s.signingParty.Receive(sender, data)

	return nil
}

// RegisterIdChangeListener is a no-op for DefaultSession
func (s *DefaultSession) RegisterIdChangeListener(func(oldId, newId string)) {}

// SigningSessionInfo is a no-op for DefaultSession
func (s *DefaultSession) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
