package utxo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	utxoutils "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/withdrawal"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	resharingConsensus "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/utxo"
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
	sessionId                    *atomic.String
	sessionLeader                core.Address
	nextSessionStartTime         time.Time
	nextSessionStartTimeConstant *atomic.Bool
	idChangeListener             func(oldId string, newId string)
	isSignSession                *atomic.Bool
	mu                           *sync.RWMutex

	parties        []p2p.Party
	sortedPartyIds tsslib.SortedPartyIDs
	self           tss.LocalSignParty
	db             db.DepositsQ
	params         session.SigningParams
	logger         *logan.Entry

	coreConnector *connector.Connector
	fetcher       *deposit.Fetcher
	client        client.Client

	signConsMechanism          consensus.Mechanism[withdrawal.UtxoWithdrawalData]
	consolidationConsMechanism consensus.Mechanism[resharingConsensus.SigningData]

	signConsParty          *consensus.Consensus[withdrawal.UtxoWithdrawalData]
	consolidationConsParty *consensus.Consensus[resharingConsensus.SigningData]

	signaturesDistributor *signing.SignaturesDistributor

	signFinalizer          *Finalizer
	consolidationFinalizer *resharingConsensus.Finalizer

	signingParty *tss.SignParty
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
		sessionId:                    atomic.NewString(sessionId),
		isSignSession:                atomic.NewBool(true),
		mu:                           &sync.RWMutex{},
		nextSessionStartTime:         params.StartTime,
		nextSessionStartTimeConstant: atomic.NewBool(true),

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

func (s *Session) WithClient(client client.Client) *Session {
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

	s.signConsMechanism = signing.NewConsensusMechanism[withdrawal.UtxoWithdrawalData](
		s.params.ChainId,
		s.db,
		withdrawal.NewBitcoinConstructor(s.client, s.self.Share.ECDSAPub.ToECDSAPubKey()),
		s.fetcher,
	)

	s.consolidationConsMechanism = resharingConsensus.NewConsensusMechanism(
		s.client,
		s.client.UtxoHelper().P2pkhAddress(s.self.Share.ECDSAPub.ToECDSAPubKey()),
		utxoutils.DefaultConsolidateOutputsParams,
	)

	return nil
}

func (s *Session) Run(ctx context.Context) error {
	if time.Until(s.nextSessionStartTime) <= 0 {
		return errors.New("target time is in the past")
	}

	for {
		// initializing required session components
		s.mu.Lock()
		s.logger = s.logger.WithField("session_id", s.Id())
		s.sessionLeader = session.DetermineLeader(s.Id(), s.sortedPartyIds)
		s.signingParty = tss.NewSignParty(s.self, s.Id(), s.logger.WithField("phase", "signing"))
		s.logger = s.logger.WithField("session_id", s.Id())
		s.signConsParty = consensus.New[withdrawal.UtxoWithdrawalData](
			consensus.LocalConsensusParty{
				SessionId: s.Id(),
				Threshold: s.self.Threshold,
				Self:      s.self.Account,
			},
			s.parties,
			s.sessionLeader,
			s.signConsMechanism,
			s.logger.WithField("phase", "consensus"),
		)
		s.signFinalizer = NewFinalizer(
			s.db, s.coreConnector, s.client,
			s.self.Share.ECDSAPub.ToECDSAPubKey(),
			s.logger.WithField("phase", "finalizing"),
			s.self.Account.CosmosAddress() == s.sessionLeader,
		)

		s.consolidationConsParty = consensus.New[resharingConsensus.SigningData](
			consensus.LocalConsensusParty{
				SessionId: s.Id(),
				Threshold: s.self.Threshold,
				Self:      s.self.Account,
			},
			s.parties,
			s.sessionLeader,
			s.consolidationConsMechanism,
			s.logger.WithField("phase", "consensus"),
		)
		s.consolidationFinalizer = resharingConsensus.NewFinalizer(
			s.client, s.self.Share.ECDSAPub.ToECDSAPubKey(),
			s.logger.WithField("phase", "finalizing"),
			s.self.Account.CosmosAddress() == s.sessionLeader,
		)
		s.signaturesDistributor = signing.NewSignaturesDistributor(
			s.Id(),
			s.parties,
			s.self,
			s.sessionLeader,
			s.logger.WithField("phase", "signatures_distributing"),
		)

		s.mu.Unlock()

		s.logger.Info(fmt.Sprintf("waiting for next session to start in %s", time.Until(s.nextSessionStartTime)))

		select {
		case <-ctx.Done():
			s.logger.Info("session cancelled")
			return nil
		case <-time.After(time.Until(s.nextSessionStartTime)):
			// nextSessionStartTime for Bitcoin session is a varying value and can be changed during the session
			s.mu.Lock()
			s.nextSessionStartTimeConstant.Store(false)
			s.nextSessionStartTime = s.nextSessionStartTime.Add(session.BoundarySigningSession)
			s.mu.Unlock()
		}

		// define the next session type
		unspentCount, err := s.client.UnspentCount()
		if err != nil {
			s.logger.WithError(err).Error("failed to get unspent count")
			s.logger.Info("starting signing session")
			s.isSignSession.Store(true)
		} else if unspentCount > s.client.ConsolidationThreshold() {
			s.logger.Info("starting consolidation session")
			s.isSignSession.Store(false)
		} else {
			s.logger.Info("starting signing session")
			s.isSignSession.Store(true)
		}

		if s.isSignSession.Load() {
			err = s.runSigningSession(ctx)
		} else {
			err = s.runConsolidationSession(ctx)
		}
		if err != nil {
			s.logger.WithError(err).Error("session error occurred")
		}
		s.logger.Info("session finished")

		s.incrementSessionId()
	}
}

func (s *Session) runSigningSession(ctx context.Context) (err error) {
	// consensus phase
	consensusCtx, consCtxCancel := context.WithTimeout(ctx, session.BoundaryConsensus)
	defer consCtxCancel()

	s.signConsParty.Run(consensusCtx)
	result, err := s.signConsParty.WaitFor()
	if err != nil {
		return errors.Wrap(err, "consensus phase error occurred")
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
		s.logger.Infof("got %d inputs to sign", signRounds)
		// signing phase
		sigs := make([]*common.SignatureData, 0, signRounds)
		for idx := range signRounds {
			currentSigData := result.SigData.ProposalData.SigData[idx]

			s.logger.Info(fmt.Sprintf("signing round %d started", idx+1))
			signingCtx, sigCtxCancel := context.WithTimeout(ctx, session.BoundarySign)

			s.signingParty.WithParties(result.Signers).WithSigningData(currentSigData).Run(signingCtx)
			signature := s.signingParty.WaitFor()
			sigCtxCancel()
			if signature == nil {
				return errors.New(fmt.Sprintf("signing phase error occurred for round %d", idx+1))
			}

			s.logger.Info(fmt.Sprintf("signing round %d finished", idx+1))
			sigs = append(sigs, signature)
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
			case <-time.After(session.BoundaryBitcoinSignRoundDelay):
			}
		}

		signatures = &tss.Signatures{Data: sigs}
		// signature distribution phase should be started not later than
		// a second after the signing phase
		distributionCtx, distributionCancel = context.WithTimeout(ctx, time.Second)
	} else {
		// party is not a signer
		// signature distribution phase should be started not later than
		// the signing phase deadline plus some extra time
		distributionCtx, distributionCancel = context.WithTimeout(ctx, getSigningPhaseDelay(signRounds)+time.Second)
	}

	// signature distribution phase
	defer distributionCancel()

	s.signaturesDistributor.
		WithSignatures(signatures).
		WithSigData(result.SigData.ProposalData.SigData).
		Run(distributionCtx)
	signatures, err = s.signaturesDistributor.WaitFor()
	if err != nil {
		return errors.Wrap(err, "signature distribution phase error occurred")
	}

	// finalization phase
	finalizerCtx, finalizerCancel := context.WithTimeout(context.Background(), session.BoundaryFinalize)
	defer finalizerCancel()

	err = s.signFinalizer.
		WithData(result.SigData).
		WithSignatures(signatures.Data).
		Finalize(finalizerCtx)
	if err != nil {
		return errors.Wrap(err, "finalizer phase error occurred")
	}

	return nil
}

func (s *Session) runConsolidationSession(ctx context.Context) error {
	// consensus phase
	consensusCtx, consCtxCancel := context.WithTimeout(ctx, session.BoundaryConsensus)
	defer consCtxCancel()

	s.consolidationConsParty.Run(consensusCtx)
	result, err := s.consolidationConsParty.WaitFor()
	if err != nil {
		return errors.Wrap(err, "consensus phase error occurred")
	}
	if result.SigData == nil {
		s.logger.Info("no data to sign in the current session")
		return nil
	}

	signRounds := len(result.SigData.ProposalData.SigData)
	s.updateNextSessionStartTime(signRounds)

	var (
		distributionCtx    context.Context
		distributionCancel context.CancelFunc
		signatures         *tss.Signatures
	)

	if result.Signers != nil {
		s.logger.Infof("got %d inputs to sign", signRounds)
		// signing phase
		sigs := make([]*common.SignatureData, 0, signRounds)
		for idx := range signRounds {
			currentSigData := result.SigData.ProposalData.SigData[idx]

			s.logger.Info(fmt.Sprintf("signing round %d started", idx+1))
			signingCtx, sigCtxCancel := context.WithTimeout(ctx, session.BoundarySign)

			s.signingParty.WithParties(result.Signers).WithSigningData(currentSigData).Run(signingCtx)
			signature := s.signingParty.WaitFor()
			sigCtxCancel()
			if signature == nil {
				return errors.New(fmt.Sprintf("signing phase error occurred for round %d", idx+1))
			}

			s.logger.Info(fmt.Sprintf("signing round %d finished", idx+1))
			sigs = append(sigs, signature)
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
			case <-time.After(session.BoundaryBitcoinSignRoundDelay):
			}
		}

		signatures = &tss.Signatures{Data: sigs}
		// signature distribution phase should be started not later than
		// a second after the signing phase
		distributionCtx, distributionCancel = context.WithTimeout(ctx, time.Second)
	} else {
		// party is not a signer
		// signature distribution phase should be started not later than
		// the signing phase deadline plus some extra time
		distributionCtx, distributionCancel = context.WithTimeout(ctx, getSigningPhaseDelay(signRounds)+time.Second)
	}

	// signature distribution phase
	defer distributionCancel()

	s.signaturesDistributor.
		WithSignatures(signatures).
		WithSigData(result.SigData.ProposalData.SigData).
		Run(distributionCtx)
	signatures, err = s.signaturesDistributor.WaitFor()
	if err != nil {
		return errors.Wrap(err, "signature distribution phase error occurred")
	}

	// finalization phase
	finalizerCtx, finalizerCancel := context.WithTimeout(context.Background(), session.BoundaryFinalize)
	defer finalizerCancel()

	txHash, err := s.consolidationFinalizer.
		WithData(result.SigData).
		WithSignatures(signatures.Data).
		Finalize(finalizerCtx)
	if err != nil {
		return errors.Wrap(err, "finalizer phase error occurred")
	}

	s.logger.Infof("consolidation has been successfully processed:  %s", txHash)

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
		var err error

		s.mu.RLock()
		if s.isSignSession.Load() {
			err = s.signConsParty.Receive(request)
		} else {
			err = s.consolidationConsParty.Receive(request)
		}
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

// updateNextSessionStartTime updates the next session start time
// based on the number of inputs required to be signed in the current session.
// By default, the next session start time at the moment of function call
// is expected at 'prevTime + tss.BoundarySigningSession'; which includes
// standard session flow: consensus -> signing (1) -> signature distribution -> finalizing
// if the number of inputs to sign is greater than 1, the next session start time
// should be recalculated to include additional signing phases and
// delays to re-setup the signing party to ensure the correct request handling
func (s *Session) updateNextSessionStartTime(inputsToSign int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSessionStartTimeConstant.Store(true)

	if inputsToSign <= 1 {
		return
	}

	// excluding included consensus, finalizing, and one signing phase
	additionalDelay := time.Duration(inputsToSign-1) * (session.BoundarySign + session.BoundaryBitcoinSignRoundDelay)
	s.nextSessionStartTime = s.nextSessionStartTime.Add(additionalDelay)
}

func (s *Session) SigningSessionInfo() *p2p.SigningSessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nextSessionStartTime *time.Time
	if s.nextSessionStartTimeConstant.Load() {
		nextSessionStartTime = &s.nextSessionStartTime
	}

	return session.ToSigningSessionInfo(
		s.Id(),
		nextSessionStartTime,
		s.self.Threshold,
		s.params.ChainId,
	)
}

func getSigningPhaseDelay(inputsToSign int) time.Duration {
	if inputsToSign <= 1 {
		return session.BoundarySign
	}

	return time.Duration(inputsToSign)*session.BoundarySign + time.Duration(inputsToSign-1)*session.BoundaryBitcoinSignRoundDelay
}
