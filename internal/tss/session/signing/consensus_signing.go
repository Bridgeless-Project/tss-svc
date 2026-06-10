package signing

import (
	"context"
	"fmt"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/bnb-chain/tss-lib/v3/common"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type ConsensusSessionResult[T consensus.SigningData] struct {
	SigningData T
	Signature   *common.SignatureData
}

type ConsensusSession[T consensus.SigningData] struct {
	id string

	parties []p2p.Party
	self    tss.LocalSignParty

	logger *logan.Entry

	signingParty          *tss.SignParty
	consensusParty        *consensus.Consensus[T]
	signaturesDistributor *SignaturesDistributor

	result *ConsensusSessionResult[T]
}

func NewConsensusSession[T consensus.SigningData](
	id string,
	self tss.LocalSignParty,
	parties []p2p.Party,
	consensusMechanism consensus.Mechanism[T],
	logger *logan.Entry,
) *ConsensusSession[T] {
	sortedPartyIds := session.SortAllParties(parties, self.Account.CosmosAddress())
	sessionId := session.GetSigningSessionIdentifier(fmt.Sprintf("%v", id))
	sessionLeader := session.DetermineLeader(sessionId, sortedPartyIds)

	return &ConsensusSession[T]{
		id: sessionId,

		parties: parties,
		self:    self,

		logger: logger,

		consensusParty: consensus.New[T](
			consensus.LocalConsensusParty{
				SessionId: sessionId,
				Threshold: self.Threshold,
				Self:      self.Account,
			},
			parties,
			sessionLeader,
			consensusMechanism,
			logger.WithField("phase", "consensus"),
		),
		signingParty: tss.NewSignParty(
			self,
			sessionId,
			logger.WithField("phase", "signing"),
		),
		signaturesDistributor: NewSignaturesDistributor(
			sessionId,
			parties,
			self,
			sessionLeader,
			logger.WithField("phase", "signatures_distributing"),
		),
	}
}

func (s *ConsensusSession[T]) Run(ctx context.Context) error {
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

	// expecting only one signature data to sign, so taking the first one
	signData := (*result.SigData).SignHashes()[0]

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
			WithSigningData(signData).
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
		WithSigData((*result.SigData).SignHashes()).
		Run(distributionCtx)
	signatures, err = s.signaturesDistributor.WaitFor()
	if err != nil {
		return errors.Wrap(err, "signature distribution phase error occurred")
	}

	s.result = &ConsensusSessionResult[T]{
		SigningData: *result.SigData,
		Signature:   signatures.Data[0],
	}

	return nil
}

func (s *ConsensusSession[T]) Result() *ConsensusSessionResult[T] {
	return s.result
}

func (s *ConsensusSession[T]) Receive(request *p2p.SubmitRequest) error {
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
	case p2p.RequestType_RT_SIGNATURE_DISTRIBUTION:
		return s.signaturesDistributor.Receive(request)
	default:
		return errors.New(fmt.Sprintf("unsupported request type %s from '%s'", request.Type, request.Sender))
	}
}

func (s *ConsensusSession[T]) Id() string {
	return s.id
}

// RegisterIdChangeListener is a no-op for ConsensusSession
func (s *ConsensusSession[T]) RegisterIdChangeListener(func(oldId, newId string)) {}

// SigningSessionInfo is a no-op for ConsensusSession
func (s *ConsensusSession[T]) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
