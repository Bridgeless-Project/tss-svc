package signing

import (
	"context"
	"fmt"
	"sync"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type SessionParams struct {
	session.Params
	SigningData []byte
}

type Session struct {
	sessionId string

	params SessionParams
	logger *logan.Entry
	wg     *sync.WaitGroup

	signingParty interface {
		Run(ctx context.Context)
		WaitFor() *common.SignatureData
		Receive(sender core.Address, data *p2p.TssData)
	}

	result *common.SignatureData
	err    error
}

func NewSession(
	self tss.LocalSignParty,
	params SessionParams,
	parties []p2p.Party,
	logger *logan.Entry,
) *Session {
	sessionId := session.GetSigningSessionIdentifier(fmt.Sprintf("%v", hexutil.Encode(params.SigningData)))
	return &Session{
		sessionId: sessionId,
		params:    params,
		wg:        &sync.WaitGroup{},
		logger:    logger,
		signingParty: tss.NewSignParty(self, sessionId, logger).
			WithSigningData(params.SigningData).
			WithParties(parties),
	}
}

func (s *Session) Run(ctx context.Context) error {
	s.logger.Info("signing session started")

	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

func (s *Session) run(ctx context.Context) {
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

func (s *Session) WaitFor() (*common.SignatureData, error) {
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

// RegisterIdChangeListener is a no-op for Session
func (s *Session) RegisterIdChangeListener(func(oldId, newId string)) {}

// SigningSessionInfo is a no-op for Session
func (s *Session) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
