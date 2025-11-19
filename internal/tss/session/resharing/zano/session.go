package zano

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ p2p.TssSession = &Session{}

type SessionParams struct {
	SessionParams  session.Params
	AssetId        string
	OwnerEthPubKey string
}

type Session struct {
	sessionId string
	self      tss.LocalSignParty
	params    SessionParams
	wg        *sync.WaitGroup

	connectedPartiesCount func() int
	parties               []p2p.Party

	client         *zano.Client
	signingParty   *tss.SignParty
	consensusParty *consensus.Consensus[SigningData]
	finalizer      *Finalizer

	resultTx string
	err      error

	logger *logan.Entry
}

func NewSession(
	self tss.LocalSignParty,
	client *zano.Client,
	params SessionParams,
	parties []p2p.Party,
	connectedPartiesCountFunc func() int,
	logger *logan.Entry,
) *Session {
	sessId := session.GetReshareSessionIdentifier(params.SessionParams.Id)
	sortedPartyIds := session.SortAllParties(parties, self.Account.CosmosAddress())
	leader := session.DetermineLeader(sessId, sortedPartyIds)

	return &Session{
		sessionId: sessId,
		self:      self,
		params:    params,
		wg:        &sync.WaitGroup{},

		connectedPartiesCount: connectedPartiesCountFunc,
		parties:               parties,

		client:       client,
		signingParty: tss.NewSignParty(self, sessId, logger.WithField("phase", "signing")),
		consensusParty: consensus.New[SigningData](
			consensus.LocalConsensusParty{
				SessionId: sessId,
				Threshold: self.Threshold,
				Self:      self.Account,
			},
			parties,
			leader,
			NewConsensusMechanism(
				params.AssetId,
				params.OwnerEthPubKey,
				client,
			),
			logger.WithField("phase", "consensus"),
		),
		finalizer: NewFinalizer(
			client,
			logger.WithField("phase", "finalizer"),
			self.Account.CosmosAddress() == leader,
		),
		logger: logger,
	}
}

func (s *Session) Run(ctx context.Context) error {
	runDelay := time.Until(s.params.SessionParams.StartTime)
	if runDelay <= 0 {
		return errors.New("target time is in the past")
	}

	s.logger.Info(fmt.Sprintf("resharing session will start in %s", runDelay))

	select {
	case <-ctx.Done():
		s.logger.Info("resharing session cancelled")
		return nil
	case <-time.After(runDelay):
		// T+1 parties required, including self
		if s.connectedPartiesCount()+1 < s.self.Threshold+1 {
			return errors.New("cannot start resharing session: not enough parties connected")
		}
	}

	s.logger.Info("resharing session started")

	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

func (s *Session) run(ctx context.Context) {
	defer s.wg.Done()

	// consensus phase
	consensusCtx, consCtxCancel := context.WithTimeout(ctx, session.BoundaryConsensus)
	defer consCtxCancel()

	s.consensusParty.Run(consensusCtx)
	result, err := s.consensusParty.WaitFor()
	if err != nil {
		s.err = errors.Wrap(err, "consensus phase error occurred")
		return
	}
	if result.Signers == nil {
		s.logger.Info("local party is not the signer in the current session")
		return
	}

	// signing phase
	signingCtx, sigCtxCancel := context.WithTimeout(ctx, session.BoundarySign)
	defer sigCtxCancel()

	s.signingParty.WithParties(result.Signers).WithSigningData(result.SigData.ProposalData.SigData).Run(signingCtx)
	signature := s.signingParty.WaitFor()
	if signature == nil {
		s.err = errors.New("signing phase error occurred ")
		return
	}

	// finalization phase
	finalizerCtx, finalizerCancel := context.WithTimeout(context.Background(), session.BoundaryFinalize)
	defer finalizerCancel()

	s.resultTx, s.err = s.finalizer.
		WithData(result.SigData).
		WithSignature(signature).
		Finalize(finalizerCtx)

	return
}

func (s *Session) Receive(request *p2p.SubmitRequest) error {
	if request == nil {
		return errors.New("nil request")
	}

	switch request.Type {
	case p2p.RequestType_RT_PROPOSAL, p2p.RequestType_RT_ACCEPTANCE, p2p.RequestType_RT_SIGN_START:
		return s.consensusParty.Receive(request)
	case p2p.RequestType_RT_SIGN:
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
	default:
		return errors.New(fmt.Sprintf("unsupported request type %s from '%s'", request.Type, request.Sender))
	}
}

func (s *Session) WaitFor() (string, error) {
	s.wg.Wait()
	return s.resultTx, s.err
}

func (s *Session) Id() string {
	return s.sessionId
}

// RegisterIdChangeListener is a no-op
func (s *Session) RegisterIdChangeListener(func(oldId string, newId string)) {}

// SigningSessionInfo is a no-op
func (s *Session) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
