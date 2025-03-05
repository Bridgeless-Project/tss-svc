package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/clients/bitcoin"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	"github.com/hyle-team/tss-svc/internal/core"
	connector "github.com/hyle-team/tss-svc/internal/core/connector"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/consensus"
	"github.com/hyle-team/tss-svc/internal/tss/finalizer"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"go.uber.org/atomic"
)

var _ p2p.TssSession = &BitcoinSigningSession{}

type BitcoinSigningSession struct {
	sessionId            *atomic.String
	nextSessionStartTime time.Time
	idChangeListener     func(oldId string, newId string)
	mu                   *sync.RWMutex

	parties []p2p.Party
	self    tss.LocalSignParty
	db      db.DepositsQ

	params SigningSessionParams
	logger *logan.Entry

	client *bitcoin.Client

	coreConnector *connector.Connector
	fetcher       *bridge.DepositFetcher
	constructor   *withdrawal.BitcoinWithdrawalConstructor

	signingParty   *tss.SignParty
	consensusParty *consensus.Consensus[withdrawal.BitcoinWithdrawalData]
	finalizer      *finalizer.BitcoinFinalizer
}

func NewBitcoinSigningSession(
	self tss.LocalSignParty,
	parties []p2p.Party,
	params SigningSessionParams,
	db db.DepositsQ,
	logger *logan.Entry,
) *BitcoinSigningSession {
	sessionId := GetConcreteSigningSessionIdentifier(params.ChainId, params.Id)

	return &BitcoinSigningSession{
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

func (s *BitcoinSigningSession) WithDepositFetcher(fetcher *bridge.DepositFetcher) *BitcoinSigningSession {
	s.fetcher = fetcher
	return s
}

func (s *BitcoinSigningSession) WithClient(client *bitcoin.Client) *BitcoinSigningSession {
	s.constructor = withdrawal.NewBitcoinConstructor(client, s.self.Share.ECDSAPub.ToECDSAPubKey())
	s.client = client
	return s
}

func (s *BitcoinSigningSession) WithCoreConnector(conn *connector.Connector) *BitcoinSigningSession {
	s.coreConnector = conn
	return s
}

func (s *BitcoinSigningSession) Run(ctx context.Context) error {
	if time.Until(s.params.StartTime) <= 0 {
		return errors.New("target time is in the past")
	}

	s.nextSessionStartTime = s.params.StartTime
	for {
		s.mu.Lock()
		s.logger = s.logger.WithField("session_id", s.Id())
		s.consensusParty = consensus.New[withdrawal.BitcoinWithdrawalData](
			consensus.LocalConsensusParty{
				SessionId: s.Id(),
				Threshold: s.self.Threshold,
				Self:      s.self.Address,
				ChainId:   s.params.ChainId,
			},
			s.parties,
			s.db,
			s.fetcher,
			s.constructor,
			s.logger.WithField("phase", "consensus"),
		)
		s.signingParty = tss.NewSignParty(s.self, s.Id(), s.logger.WithField("phase", "signing"))
		s.finalizer = finalizer.NewBitcoinFinalizer(s.db, s.coreConnector, s.client, s.self.Share.ECDSAPub.ToECDSAPubKey(), s.logger.WithField("phase", "finalizing"))
		s.mu.Unlock()

		s.logger.Info(fmt.Sprintf("waiting for next signing session %s to start in %s", s.Id(), time.Until(s.nextSessionStartTime)))

		select {
		case <-ctx.Done():
			s.logger.Info("signing session cancelled")
			return nil
		case <-time.After(time.Until(s.nextSessionStartTime)):
			// nextSessionStartTime for Bitcoin session is a varying value and can be changed during the session
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

func (s *BitcoinSigningSession) runSession(ctx context.Context) error {
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
				s.updateNextSessionStartTime(len(result.SigData.ProposalData.SigData))
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
	signRounds := len(result.SigData.ProposalData.SigData)
	s.updateNextSessionStartTime(signRounds)
	if err = s.db.UpdateStatus(result.SigData.DepositIdentifier(), types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSING); err != nil {
		return errors.Wrap(err, "failed to update deposit status")
	}
	if result.Signers == nil {
		s.logger.Info("local party is not the signer in the current session")
		return nil
	}

	s.logger.Infof("got %d inputs to sign", signRounds)
	// signing phase
	signatures := make([]*common.SignatureData, 0, signRounds)
	for idx := range signRounds {
		currentSigData := result.SigData.ProposalData.SigData[idx]

		s.logger.Info(fmt.Sprintf("signing round %d started", idx+1))
		signingCtx, sigCtxCancel := context.WithTimeout(ctx, tss.BoundarySign)

		s.signingParty.WithParties(result.Signers).WithSigningData(currentSigData).Run(signingCtx)
		signature := s.signingParty.WaitFor()
		sigCtxCancel()
		if signature == nil {
			return errors.New(fmt.Sprintf("signing phase error occurred for round %d", idx+1))
		}

		s.logger.Info(fmt.Sprintf("signing round %d finished", idx+1))
		signatures = append(signatures, signature)
		if idx+1 == signRounds {
			break
		}

		s.mu.Lock()
		s.signingParty = tss.NewSignParty(s.self, s.Id(), s.logger.WithField("phase", "signing"))
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			s.logger.Info("signing session cancelled")
			return nil
		case <-time.After(tss.BoundaryBitcoinSingRoundDelay):
		}
	}

	// finalization phase
	finalizerCtx, finalizerCancel := context.WithTimeout(ctx, tss.BoundaryFinalize)
	defer finalizerCancel()

	err = s.finalizer.
		WithData(result.SigData).
		WithSignatures(signatures).
		WithLocalPartyProposer(s.self.Address == result.Proposer).
		Finalize(finalizerCtx)
	if err != nil {
		return errors.Wrap(err, "finalizer phase error occurred")
	}

	return nil
}

func (s *BitcoinSigningSession) Id() string {
	return s.sessionId.Load()
}

func (s *BitcoinSigningSession) incrementSessionId() {
	prevSessionId := s.Id()
	nextSessionId := IncrementSessionIdentifier(prevSessionId)
	s.sessionId.Store(nextSessionId)
	s.idChangeListener(prevSessionId, nextSessionId)
}

func (s *BitcoinSigningSession) Receive(request *p2p.SubmitRequest) error {
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

func (s *BitcoinSigningSession) RegisterIdChangeListener(f func(oldId string, newId string)) {
	s.idChangeListener = f
}

// updateNextSessionStartTime updates the next session start time
// based on the number of inputs required to be signed in the current session.
// By default, the next session start time at the moment of function call
// is expected at 'prevTime + tss.BoundarySigningSession'; which includes
// standard session flow: consensus -> signing (1) -> finalizing
// if the number of inputs to sign is greater than 1, the next session start time
// should be recalculated to include additional signing phases and
// delays to re-setup the signing party to ensure the correct request handling
func (s *BitcoinSigningSession) updateNextSessionStartTime(inputsToSign int) {
	if inputsToSign <= 1 {
		return
	}

	// excluding included consensus, finalizing, and one signing phase
	additionalDelay := time.Duration(inputsToSign-1) * (tss.BoundarySign + tss.BoundaryBitcoinSingRoundDelay)
	s.nextSessionStartTime = s.nextSessionStartTime.Add(additionalDelay)
}

func (s *BitcoinSigningSession) Info() (*p2p.SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	startTime := s.nextSessionStartTime.Unix()
	id, err := GetSessionId(s.sessionId.Load())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session id")
	}

	return &p2p.SessionInfo{
		Id:                   id,
		NextSessionStartTime: startTime,
		Mode:                 p2p.SessionMode_SM_SIGN,
	}, nil
}

func (s *BitcoinSigningSession) ChainID() string {
	return s.params.ChainId
}
