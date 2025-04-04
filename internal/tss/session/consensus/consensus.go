package consensus

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

const msgsCapacity = 100

type consensusMsg struct {
	Sender core.Address
	Type   p2p.RequestType
	Data   *anypb.Any
}

type LocalConsensusParty struct {
	Self      core.Account
	SessionId string
	Threshold int
}

type SigningSessionData[T SigningData] struct {
	SigData  *T
	Signers  []p2p.Party
	Proposer core.Address
}

func New[T SigningData](
	party LocalConsensusParty,
	parties []p2p.Party,
	mechanism Mechanism[T],
	logger *logan.Entry,
) *Consensus[T] {
	partiesMap := make(map[core.Address]p2p.Party, len(parties))
	for _, p := range parties {
		partiesMap[p.CoreAddress] = p
	}

	return &Consensus[T]{
		mechanism: mechanism,
		parties:   partiesMap,

		proposalBroadcaster: p2p.NewReliableBroadcaster[T](
			party.SessionId,
			parties,
			party.Self,
			party.Threshold,
			p2p.RequestType_RT_PROPOSAL,
			logger.WithField("component", "proposal_broadcaster"),
		),
		signStartBroadcaster: p2p.NewReliableBroadcaster[SignStartData](
			party.SessionId,
			parties,
			party.Self,
			party.Threshold,
			p2p.RequestType_RT_SIGN_START,
			logger.WithField("component", "sign_start_broadcaster"),
		),
		broadcaster: p2p.NewBroadcaster(parties, logger.WithField("component", "broadcaster")),

		self:      party.Self,
		sessionId: party.SessionId,
		threshold: party.Threshold,

		logger: logger.WithField("session_id", party.SessionId),

		wg:   &sync.WaitGroup{},
		msgs: make(chan consensusMsg, msgsCapacity),
	}
}

type Consensus[T SigningData] struct {
	mechanism Mechanism[T]
	parties   map[core.Address]p2p.Party

	proposalBroadcaster  *p2p.ReliableBroadcaster[T]
	signStartBroadcaster *p2p.ReliableBroadcaster[SignStartData]
	broadcaster          *p2p.Broadcaster

	self      core.Account
	sessionId string
	threshold int

	logger *logan.Entry

	proposer core.Address
	wg       *sync.WaitGroup
	ended    atomic.Bool
	msgs     chan consensusMsg

	result struct {
		sigData *T
		signers []p2p.Party
		err     error
	}
}

func (c *Consensus[T]) Receive(request *p2p.SubmitRequest) error {
	if request == nil {
		return errors.New("nil request")
	}

	if request.SessionId != c.sessionId {
		return errors.New(fmt.Sprintf("session id mismatch: expected '%s', got '%s'", c.sessionId, request.SessionId))
	}

	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}

	if _, exists := c.parties[sender]; !exists {
		return errors.New(fmt.Sprintf("unknown sender '%s'", sender))
	}

	switch request.Type {
	case p2p.RequestType_RT_PROPOSAL:
		data := &p2p.ReliableBroadcastData{}
		if err = request.Data.UnmarshalTo(data); err != nil {
			return errors.Wrap(err, "failed to unmarshal reliable broadcast data")
		}
		roundMsg, err := p2p.DecodeRoundMessage[T](data.GetRoundMsg())
		if err != nil {
			return errors.Wrap(err, "failed to decode round message")
		}
		if roundMsg.Round == 1 {
			c.msgs <- consensusMsg{
				Sender: sender,
				Type:   request.Type,
				Data:   request.Data,
			}
			return nil
		}

		return c.proposalBroadcaster.Receive(p2p.ReliableBroadcastMsg[T]{
			Sender: sender,
			Msg:    roundMsg,
		})
	case p2p.RequestType_RT_SIGN_START:
		data := &p2p.ReliableBroadcastData{}
		if err = request.Data.UnmarshalTo(data); err != nil {
			return errors.Wrap(err, "failed to unmarshal reliable broadcast data")
		}
		roundMsg, err := p2p.DecodeRoundMessage[SignStartData](data.GetRoundMsg())
		if err != nil {
			return errors.Wrap(err, "failed to decode round message")
		}
		if roundMsg.Round == 1 {
			c.msgs <- consensusMsg{
				Sender: sender,
				Type:   request.Type,
				Data:   request.Data,
			}
			return nil
		}

		return c.signStartBroadcaster.Receive(p2p.ReliableBroadcastMsg[SignStartData]{
			Sender: sender,
			Msg:    roundMsg,
		})
	case p2p.RequestType_RT_ACCEPTANCE:
		c.msgs <- consensusMsg{
			Sender: sender,
			Type:   request.Type,
			Data:   request.Data,
		}
	default:
		return errors.New(fmt.Sprintf("unsupported request type %s from '%s')", request.Type, sender))
	}

	return nil
}

func (c *Consensus[T]) Run(ctx context.Context) {
	c.proposer = c.determineProposer()
	c.logger.Info(fmt.Sprintf("starting consensus with proposer: %s", c.proposer))

	c.wg.Add(1)
	if c.proposer == c.self.CosmosAddress() {
		go c.propose(ctx)
	} else {
		go c.accept(ctx)
	}
}

func (c *Consensus[T]) WaitFor() (result SigningSessionData[T], err error) {
	c.wg.Wait()
	c.ended.Store(true)
	c.logger.Info("consensus finished")

	return SigningSessionData[T]{
		SigData:  c.result.sigData,
		Signers:  c.result.signers,
		Proposer: c.proposer,
	}, c.result.err
}

func (c *Consensus[T]) determineProposer() core.Address {
	partyIds := make([]*tss.PartyID, len(c.parties)+1)
	idx := 0
	for _, party := range c.parties {
		partyIds[idx] = party.Identifier()
		idx++
	}
	partyIds[idx] = c.self.CosmosAddress().PartyIdentifier()

	sortedIds := tss.SortPartyIDs(partyIds)

	generator := deterministicRandSource(c.sessionId)
	proposerIdx := int(generator.Uint64() % uint64(sortedIds.Len()))

	return core.AddrFromPartyId(sortedIds[proposerIdx])
}

func deterministicRandSource(sessionId string) rand.Source {
	seed := sha256.Sum256([]byte(sessionId))
	return rand.NewChaCha8(seed)
}
