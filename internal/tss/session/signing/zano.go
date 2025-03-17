package signing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/zano"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	"github.com/hyle-team/tss-svc/internal/core"
	connector "github.com/hyle-team/tss-svc/internal/core/connector"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/consensus"
	"github.com/hyle-team/tss-svc/internal/tss/finalizer"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"go.uber.org/atomic"
)

var _ p2p.TssSession = &ZanoSession{}

type ZanoSession struct {
	sessionId            *atomic.String
	idChangeListener     func(oldId string, newId string)
	mu                   *sync.RWMutex
	nextSessionStartTime time.Time

	parties []p2p.Party
	self    tss.LocalSignParty
	db      db.DepositsQ
	params  session.SigningSessionParams
	logger  *logan.Entry

	client        *zano.Client
	coreConnector *connector.Connector
	fetcher       *bridge.DepositFetcher

	mechanism consensus.Mechanism[withdrawal.ZanoWithdrawalData]

	signingParty   *tss.SignParty
	consensusParty *consensus.Consensus[withdrawal.ZanoWithdrawalData]
	finalizer      *finalizer.ZanoFinalizer
}

func NewZanoSession(
	self tss.LocalSignParty,
	parties []p2p.Party,
	params session.SigningSessionParams,
	db db.DepositsQ,
	logger *logan.Entry,
) *ZanoSession {
	sessionId := session.GetConcreteSigningSessionIdentifier(params.ChainId, params.Id)

	return &ZanoSession{
		sessionId:            atomic.NewString(sessionId),
		mu:                   &sync.RWMutex{},
		nextSessionStartTime: params.StartTime,

		parties: parties,
		self:    self,
		db:      db,

		params: params,
		logger: logger,
	}
}

func (s *ZanoSession) WithDepositFetcher(fetcher *bridge.DepositFetcher) *ZanoSession {
	s.fetcher = fetcher
	return s
}

func (s *ZanoSession) WithCoreConnector(conn *connector.Connector) *ZanoSession {
	s.coreConnector = conn
	return s
}

func (s *ZanoSession) WithClient(client *zano.Client) *ZanoSession {
	s.client = client
	return s
}

// Build is a method that should be called before Run to prepare the session for execution.
func (s *ZanoSession) Build() error {
	if s.fetcher == nil {
		return errors.New("deposit fetcher is not set")
	}
	if s.client == nil {
		return errors.New("blockchain client is not set")
	}
	if s.coreConnector == nil {
		return errors.New("core connector is not set")
	}

	s.mechanism = withdrawal.NewConsensusMechanism[withdrawal.ZanoWithdrawalData](
		s.params.ChainId,
		s.db,
		withdrawal.NewZanoConstructor(s.client),
		s.fetcher,
	)

	return nil
}

func (s *ZanoSession) Run(ctx context.Context) error {
	if time.Until(s.params.StartTime) <= 0 {
		return errors.New("target time is in the past")
	}

	for {
		s.mu.Lock()
		s.logger = s.logger.WithField("session_id", s.Id())
		s.consensusParty = consensus.New[withdrawal.ZanoWithdrawalData](
			consensus.LocalConsensusParty{
				SessionId: s.Id(),
				Threshold: s.self.Threshold,
				Self:      s.self.Address,
			},
			s.parties,
			s.mechanism,
			s.logger.WithField("phase", "consensus"),
		)
		s.signingParty = tss.NewSignParty(s.self, s.Id(), s.logger.WithField("phase", "signing"))
		s.finalizer = finalizer.NewZanoFinalizer(s.db, s.coreConnector, s.client, s.logger.WithField("phase", "finalizing"))
		s.mu.Unlock()

		s.logger.Info(fmt.Sprintf("waiting for next signing session %s to start in %s", s.Id(), time.Until(s.nextSessionStartTime)))

		select {
		case <-ctx.Done():
			s.logger.Info("signing session cancelled")
			return nil
		case <-time.After(time.Until(s.nextSessionStartTime)):
			s.nextSessionStartTime = s.nextSessionStartTime.Add(tss.BoundarySigningSession)
		}

		s.logger.Info(fmt.Sprintf("signing session %s started", s.Id()))
		if err := s.runSession(ctx); err != nil {
			s.logger.WithError(err).Error("failed to run signing session")
		}
		s.logger.Info(fmt.Sprintf("signing session %s finished", s.Id()))

		s.incrementSessionId()
	}
}

func (s *ZanoSession) runSession(ctx context.Context) error {
	// consensus phase
	consensusCtx, consCtxCancel := context.WithTimeout(ctx, tss.BoundaryConsensus)
	defer consCtxCancel()

	s.consensusParty.Run(consensusCtx)
	result, err := s.consensusParty.WaitFor()
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			return errors.Wrap(err, "consensus phase error occurred")

		}
		if err = ctx.Err(); err != nil {
			s.logger.Info("session cancelled")
			return nil
		}
		if err = consensusCtx.Err(); err != nil {
			if result.SigData != nil {
				s.logger.Info("local party is not the signer in the current session")
			} else {
				s.logger.Info("consensus phase timeout")
			}
			return nil
		}
	}
	if result.SigData == nil {
		s.logger.Info("no data to sign in the current session")
		return nil
	}
	if err = s.db.UpdateStatus(result.SigData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSING); err != nil {
		return errors.Wrap(err, "failed to update deposit status")
	}
	if result.Signers == nil {
		s.logger.Info("local party is not the signer in the current session")
		return nil
	}

	// signing phase
	signingCtx, sigCtxCancel := context.WithTimeout(ctx, tss.BoundarySign)
	defer sigCtxCancel()

	s.signingParty.WithParties(result.Signers).WithSigningData(result.SigData.ProposalData.SigData).Run(signingCtx)
	signature := s.signingParty.WaitFor()
	if signature == nil {
		return errors.New("signing phase error occurred")
	}

	// finalization phase
	finalizerCtx, finalizerCancel := context.WithTimeout(ctx, tss.BoundaryFinalize)
	defer finalizerCancel()

	err = s.finalizer.
		WithData(result.SigData).
		WithSignature(signature).
		WithLocalPartyProposer(s.self.Address == result.Proposer).
		Finalize(finalizerCtx)
	if err != nil {
		return errors.Wrap(err, "finalizer phase error occurred")
	}

	return nil
}

func (s *ZanoSession) Id() string {
	return s.sessionId.Load()
}

func (s *ZanoSession) incrementSessionId() {
	prevSessionId := s.Id()
	nextSessionId := session.IncrementSessionIdentifier(prevSessionId)
	s.sessionId.Store(nextSessionId)
	s.idChangeListener(prevSessionId, nextSessionId)
}

func (s *ZanoSession) Receive(request *p2p.SubmitRequest) error {
	if request == nil {
		return errors.New("nil request")
	}

	switch request.Type {
	case p2p.RequestType_RT_PROPOSAL, p2p.RequestType_RT_ACCEPTANCE, p2p.RequestType_RT_SIGN_START:
		s.mu.RLock()
		err := s.consensusParty.Receive(request)
		s.mu.RUnlock()

		return err
	case p2p.RequestType_RT_SIGN:
		data := &p2p.TssData{}
		if err := request.Data.UnmarshalTo(data); err != nil {
			return errors.Wrap(err, "failed to unmarshal TSS request signingData")
		}

		sender, err := core.AddressFromString(request.Sender)
		if err != nil {
			return errors.Wrap(err, "failed to parse sender address")
		}

		s.mu.RLock()
		s.signingParty.Receive(sender, data)
		s.mu.RUnlock()

		return nil
	default:
		return errors.New(fmt.Sprintf("unsupported request type %s from '%s'", request.Type, request.Sender))
	}
}

func (s *ZanoSession) RegisterIdChangeListener(f func(oldId string, newId string)) {
	s.idChangeListener = f
}

func (s *ZanoSession) SigningSessionInfo() *p2p.SigningSessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return session.ToSigningSessionInfo(
		s.Id(),
		&s.nextSessionStartTime,
		s.self.Threshold,
		s.params.ChainId,
	)
}
