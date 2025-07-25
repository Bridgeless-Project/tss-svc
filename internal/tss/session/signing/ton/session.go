package ton

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/ton"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/bnb-chain/tss-lib/v2/common"
	tsslib "github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"go.uber.org/atomic"
)

var _ p2p.TssSession = &Session{}

type Session struct {
	sessionId            *atomic.String
	sessionLeader        core.Address
	idChangeListener     func(oldId string, newId string)
	mu                   *sync.RWMutex
	nextSessionStartTime time.Time

	parties        []p2p.Party
	sortedPartyIds tsslib.SortedPartyIDs

	self   tss.LocalSignParty
	db     db.DepositsQ
	params session.SigningParams
	logger *logan.Entry

	coreConnector *connector.Connector
	fetcher       *deposit.Fetcher
	client        *ton.Client

	mechanism consensus.Mechanism[withdrawal.TonWithdrawalData]

	signingParty          *tss.SignParty
	consensusParty        *consensus.Consensus[withdrawal.TonWithdrawalData]
	signaturesDistributor *signing.SignaturesDistributor
	finalizer             *Finalizer
}

func NewSession(
	self tss.LocalSignParty,
	parties []p2p.Party,
	params session.SigningParams,
	db db.DepositsQ,
	logger *logan.Entry,
) *Session {
	sessionId := session.GetConcreteSigningSessionIdentifier(params.ChainId, params.Id)

	return &Session{
		sessionId:            atomic.NewString(sessionId),
		mu:                   &sync.RWMutex{},
		nextSessionStartTime: params.StartTime,

		parties:        parties,
		self:           self,
		db:             db,
		sortedPartyIds: session.SortAllParties(parties, self.Account.CosmosAddress()),

		params: params,
		logger: logger,
	}
}

func (s *Session) WithDepositFetcher(fetcher *deposit.Fetcher) *Session {
	s.fetcher = fetcher
	return s
}

func (s *Session) WithClient(client *ton.Client) *Session {
	s.client = client
	return s
}

func (s *Session) WithCoreConnector(conn *connector.Connector) *Session {
	s.coreConnector = conn
	return s
}

// Build is a method that should be called before Run to prepare the session for execution.
func (s *Session) Build() error {
	if s.fetcher == nil {
		return errors.New("deposit fetcher is not set")
	}
	if s.client == nil {
		return errors.New("blockchain client is not set")
	}
	if s.coreConnector == nil {
		return errors.New("core connector is not set")
	}

	s.mechanism = signing.NewConsensusMechanism[withdrawal.TonWithdrawalData](
		s.params.ChainId,
		s.db,
		withdrawal.NewTonConstructor(s.client),
		s.fetcher,
	)

	return nil
}

func (s *Session) Run(ctx context.Context) error {
	if time.Until(s.nextSessionStartTime) <= 0 {
		return errors.New("target time is in the past")
	}

	for {
		s.mu.Lock()
		s.logger = s.logger.WithField("session_id", s.Id())
		s.sessionLeader = session.DetermineLeader(s.Id(), s.sortedPartyIds)
		s.consensusParty = consensus.New[withdrawal.TonWithdrawalData](
			consensus.LocalConsensusParty{
				SessionId: s.Id(),
				Threshold: s.self.Threshold,
				Self:      s.self.Account,
			},
			s.parties,
			s.sessionLeader,
			s.mechanism,
			s.logger.WithField("phase", "consensus"),
		)
		s.signingParty = tss.NewSignParty(s.self, s.Id(), s.logger.WithField("phase", "signing"))
		s.signaturesDistributor = signing.NewSignaturesDistributor(
			s.Id(),
			s.parties,
			s.self,
			s.sessionLeader,
			s.logger.WithField("phase", "signatures_distributing"),
		)
		s.finalizer = NewFinalizer(
			s.db,
			s.coreConnector,
			s.logger.WithField("phase", "finalizing"),
			s.self.Account.CosmosAddress() == s.sessionLeader,
		)
		s.mu.Unlock()

		s.logger.Info(fmt.Sprintf("waiting for next signing session %s to start in %s", s.Id(), time.Until(s.nextSessionStartTime)))

		select {
		case <-ctx.Done():
			s.logger.Info("signing session cancelled")
			return nil
		case <-time.After(time.Until(s.nextSessionStartTime)):
			s.nextSessionStartTime = s.nextSessionStartTime.Add(session.BoundarySigningSession)
		}

		s.logger.Info(fmt.Sprintf("signing session %s started", s.Id()))
		if err := s.runSession(ctx); err != nil {
			s.logger.WithError(err).Error("failed to run signing session")
		}
		s.logger.Info(fmt.Sprintf("signing session %s finished", s.Id()))

		s.incrementSessionId()
	}
}

func (s *Session) runSession(ctx context.Context) (err error) {
	// consensus phase
	consensusCtx, consCtxCancel := context.WithTimeout(ctx, session.BoundaryConsensus)
	defer consCtxCancel()

	s.consensusParty.Run(consensusCtx)
	result, err := s.consensusParty.WaitFor()
	if err != nil {
		return errors.Wrap(err, "consensus phase error occurred")
	}
	if result.SigData == nil {
		s.logger.Info("no data to sign in the current session")
		return nil
	}

	if err = s.db.UpdateStatus(result.SigData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSING); err != nil {
		return errors.Wrap(err, "failed to update deposit status")
	}
	defer func() {
		// compensating status update in case of error
		if err != nil {
			_ = s.db.UpdateStatus(result.SigData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING)
		}
	}()

	var (
		distributionCtx    context.Context
		distributionCancel context.CancelFunc
		signatures         *tss.Signatures
	)
	if result.Signers != nil {
		// the party takes part in a signing process
		signingCtx, sigCtxCancel := context.WithTimeout(ctx, session.BoundarySign)
		defer sigCtxCancel()

		s.signingParty.
			WithParties(result.Signers).
			WithSigningData(result.SigData.ProposalData.SigData).
			Run(signingCtx)
		signature := s.signingParty.WaitFor()
		if signature == nil {
			return errors.New("signing phase error occurred")
		}

		signatures = &tss.Signatures{
			Data: []*common.SignatureData{signature},
		}

		// signature distribution phase should be started not later than
		// a second after the signing phase
		distributionCtx, distributionCancel = context.WithTimeout(ctx, time.Second)
	} else {
		// party is not a signer
		// signature distribution phase should be started not later than
		// the signing phase deadline plus some extra time
		distributionCtx, distributionCancel = context.WithTimeout(ctx, session.BoundarySign+time.Second)
	}

	// signature distribution phase
	defer distributionCancel()

	s.signaturesDistributor.
		WithSignatures(signatures).
		WithSigData([][]byte{result.SigData.ProposalData.SigData}).
		Run(distributionCtx)
	signatures, err = s.signaturesDistributor.WaitFor()
	if err != nil {
		return errors.Wrap(err, "signature distribution phase error occurred")
	}

	// finalization phase
	finalizerCtx, finalizerCancel := context.WithTimeout(context.Background(), session.BoundaryFinalize)
	defer finalizerCancel()

	err = s.finalizer.
		WithData(result.SigData).
		WithSignature(signatures.Data[0]).
		Finalize(finalizerCtx)
	if err != nil {
		return errors.Wrap(err, "finalizer phase error occurred")
	}

	return nil
}

func (s *Session) Id() string {
	return s.sessionId.Load()
}

func (s *Session) incrementSessionId() {
	prevSessionId := s.Id()
	nextSessionId := session.IncrementSessionIdentifier(prevSessionId)
	s.sessionId.Store(nextSessionId)
	s.idChangeListener(prevSessionId, nextSessionId)
}

func (s *Session) Receive(request *p2p.SubmitRequest) error {
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
	case p2p.RequestType_RT_SIGNATURE_DISTRIBUTION:
		s.mu.RLock()
		err := s.signaturesDistributor.Receive(request)
		s.mu.RUnlock()

		return err
	default:
		return errors.New(fmt.Sprintf("unsupported request type %s from '%s'", request.Type, request.Sender))
	}
}

func (s *Session) RegisterIdChangeListener(f func(oldId string, newId string)) {
	s.idChangeListener = f
}

func (s *Session) SigningSessionInfo() *p2p.SigningSessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return session.ToSigningSessionInfo(
		s.Id(),
		&s.nextSessionStartTime,
		s.self.Threshold,
		s.params.ChainId,
	)
}
