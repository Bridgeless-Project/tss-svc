package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"go.uber.org/atomic"
)

var _ p2p.TssSession = &EvmSigningSession{}

type EvmSigningSession struct {
	sessionId        atomic.String
	idChangeListener func(oldId string, newId string)
	mu               *sync.RWMutex

	self tss.LocalSignParty

	params SigningSessionParams
	logger *logan.Entry

	signingParty interface {
		Run(ctx context.Context)
		WaitFor() *common.SignatureData
		Receive(sender core.Address, data *p2p.TssData)
	}

	consensusParty interface {
		Run(ctx context.Context)
		WaitFor() *common.SignatureData
		Receive(sender core.Address, data *p2p.TssData)
	}
}

func (s *EvmSigningSession) Run(ctx context.Context) error {
	runDelay := time.Until(s.params.StartTime)
	if runDelay <= 0 {
		return errors.New("target time is in the past")
	}

	s.logger.Info(fmt.Sprintf("signing session will start in %s", runDelay))

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(runDelay):
	}

	// init first sigparty
	// init first consensus

	sessionStartTime := time.Now()
	for {
		s.logger.Info(fmt.Sprintf("signing session %s started", s.sessionId.Load()))
		// consensusCtx, cancel := context.WithTimeout(ctx, tss.BoundaryConsensus)

		// start consensus
		// check consensus params
		// start sigparty

		s.logger.Info(fmt.Sprintf("signing session %s ended, waiting for the next one", s.sessionId.Load()))

		s.mu.Lock()
		nextSessionId := IncrementSessionIdentifier(s.sessionId.Load())
		s.sessionId.Store(nextSessionId)
		s.signingParty = tss.NewSignParty(s.self, nextSessionId, s.logger)
		// TODO: init next consensus
		s.mu.Unlock()

		nextSessionStart := sessionStartTime.Add(tss.BoundarySigningSession)
		select {
		case <-ctx.Done():
			s.logger.Info("signing session cancelled")
			return ctx.Err()
		case <-time.After(time.Until(nextSessionStart)):
			sessionStartTime = nextSessionStart
		}
	}
}

func (s *EvmSigningSession) Id() string {
	return s.sessionId.Load()
}

func (s *EvmSigningSession) Receive(request *p2p.SubmitRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s *EvmSigningSession) RegisterIdChangeListener(f func(oldId string, newId string)) {
	s.idChangeListener = f
}
