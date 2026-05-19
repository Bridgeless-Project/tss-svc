package test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	chain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/test"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	tssProtocols "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"go.uber.org/atomic"
)

var _ p2p.TssSession = &Session{}

type MockSigningData struct {
	Message string
	Hash    []byte
}

func NewMockSigningData() MockSigningData {
	hash := sha256.Sum256([]byte(chain.MockSigningMessage))
	return MockSigningData{
		Message: chain.MockSigningMessage,
		Hash:    hash[:],
	}
}

func (d MockSigningData) HashString() string {
	hash := sha256.Sum256(append([]byte(d.Message), d.Hash...))
	return fmt.Sprintf("%x", hash)
}

type mockConsensusMechanism struct{}

func (m mockConsensusMechanism) FormProposalData() (*MockSigningData, error) {
	data := NewMockSigningData()
	return &data, nil
}

func (m mockConsensusMechanism) VerifyProposedData(data MockSigningData) error {
	expected := NewMockSigningData()
	if data.Message != expected.Message {
		return errors.New("mock message does not match expected")
	}
	if !bytes.Equal(data.Hash, expected.Hash) {
		return errors.New("mock hash does not match expected")
	}

	return nil
}

type Session struct {
	sessionId            *atomic.String
	sessionLeader        core.Address
	idChangeListener     func(oldId string, newId string)
	mu                   *sync.RWMutex
	nextSessionStartTime time.Time

	parties []p2p.Party
	self    tss.LocalSignParty
	params  session.SigningParams
	logger  *logan.Entry

	signingParty          tss.SignParty
	consensusParty        *consensus.Consensus[MockSigningData]
	signaturesDistributor *signing.SignaturesDistributor
}

func NewSession(
	self tss.LocalSignParty,
	parties []p2p.Party,
	params session.SigningParams,
	_ db.DepositsQ,
	logger *logan.Entry,
) *Session {
	sessionId := session.GetConcreteSigningSessionIdentifier(params.ChainId, params.Id)

	return &Session{
		sessionId:            atomic.NewString(sessionId),
		mu:                   &sync.RWMutex{},
		nextSessionStartTime: params.StartTime,
		parties:              parties,
		self:                 self,
		params:               params,
		logger:               logger,
	}
}

func (s *Session) Build() error {
	if s.self.FrostShare == nil {
		return errors.New("test signing session requires FROST share")
	}

	if _, err := tss.FrostPubKey(s.self.FrostShare); err != nil {
		return errors.Wrap(err, "invalid FROST share")
	}

	return nil
}

func (s *Session) Run(ctx context.Context) error {
	if time.Until(s.nextSessionStartTime) <= 0 {
		return errors.New("target time is in the past")
	}

	for {
		s.prepareRound()

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

func (s *Session) prepareRound() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger = s.logger.WithField("session_id", s.Id())
	s.sessionLeader = session.DetermineLeader(s.Id(), session.SortAllParties(s.parties, s.self.Account.CosmosAddress()))
	s.consensusParty = consensus.New[MockSigningData](
		consensus.LocalConsensusParty{
			SessionId: s.Id(),
			Threshold: s.self.Threshold,
			Self:      s.self.Account,
		},
		s.parties,
		s.sessionLeader,
		mockConsensusMechanism{},
		s.logger.WithField("phase", "consensus"),
	)
	s.signingParty = tssProtocols.SelectSignByShare(s.self, s.Id(), s.logger.WithField("phase", "signing"))
	s.signaturesDistributor = signing.NewSignaturesDistributor(
		s.Id(),
		s.parties,
		s.self,
		s.sessionLeader,
		s.logger.WithField("phase", "signatures_distributing"),
	)
}

func (s *Session) runSession(ctx context.Context) error {
	consensusCtx, consCtxCancel := context.WithTimeout(ctx, session.BoundaryFrostConsensus)
	defer consCtxCancel()

	s.consensusParty.Run(consensusCtx)
	result, err := s.consensusParty.WaitFor()
	if err != nil {
		return errors.Wrap(err, "consensus phase error occurred")
	}
	if result.SigData == nil {
		s.logger.Info("no mock data to sign in the current session")
		return nil
	}

	var (
		distributionCtx    context.Context
		distributionCancel context.CancelFunc
		signatures         *tss.Signatures
	)
	if result.Signers != nil {
		signingCtx, sigCtxCancel := context.WithTimeout(ctx, session.BoundarySign)
		defer sigCtxCancel()

		s.signingParty.
			WithParties(result.Signers).
			WithSigningData(result.SigData.Hash).
			Run(signingCtx)
		signature := s.signingParty.WaitFor()
		if signature == nil {
			return errors.New("signing phase error occurred")
		}

		signatures = &tss.Signatures{
			Data: []*common.SignatureData{signature},
		}

		distributionCtx, distributionCancel = context.WithTimeout(ctx, time.Second)
	} else {
		distributionCtx, distributionCancel = context.WithTimeout(ctx, session.BoundarySign+time.Second)
	}
	defer distributionCancel()

	s.signaturesDistributor.
		WithSignatures(signatures).
		WithSigData([][]byte{result.SigData.Hash}).
		Run(distributionCtx)
	signatures, err = s.signaturesDistributor.WaitFor()
	if err != nil {
		return errors.Wrap(err, "signature distribution phase error occurred")
	}

	return s.printResult(*result.SigData, signatures)
}

func (s *Session) printResult(data MockSigningData, signatures *tss.Signatures) error {
	if signatures == nil || len(signatures.Data) == 0 || signatures.Data[0] == nil {
		return errors.New("missing mock signing result")
	}

	pubKey, err := tss.FrostPubKey(s.self.FrostShare)
	if err != nil {
		return errors.Wrap(err, "failed to get FROST public key")
	}

	signature := signatures.Data[0]
	output := struct {
		Protocol  string `json:"protocol"`
		Curve     string `json:"curve"`
		Message   string `json:"message"`
		Hash      string `json:"hash"`
		PubKey    string `json:"pubkey"`
		Signature string `json:"signature"`
		Verified  bool   `json:"verified"`
	}{
		Protocol:  "frost",
		Curve:     "secp256k1",
		Message:   data.Message,
		Hash:      hexutil.Encode(data.Hash),
		PubKey:    hexutil.Encode(pubKey),
		Signature: hexutil.Encode(signature.Signature),
		Verified:  tss.VerifyFrost(pubKey, data.Hash, signature),
	}

	if !output.Verified {
		return errors.New("mock FROST signature verification failed")
	}

	raw, err := json.Marshal(output)
	if err != nil {
		return errors.Wrap(err, "failed to marshal mock signing result")
	}

	fmt.Println(string(raw))
	return nil
}

func (s *Session) Id() string {
	return s.sessionId.Load()
}

func (s *Session) incrementSessionId() {
	prevSessionId := s.Id()
	nextSessionId := session.IncrementSessionIdentifier(prevSessionId)
	s.sessionId.Store(nextSessionId)
	if s.idChangeListener != nil {
		s.idChangeListener(prevSessionId, nextSessionId)
	}
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
