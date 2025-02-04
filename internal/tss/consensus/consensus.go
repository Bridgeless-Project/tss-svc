package consensus

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/withdrawal"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/db"
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
	Self      core.Address
	SessionId string
	Threshold int
	ChainId   string
}

func New[T withdrawal.DepositSigningData](
	party LocalConsensusParty,
	parties []p2p.Party,
	db db.DepositsQ,
	processor *bridge.DepositFetcher,
	constructor withdrawal.Constructor[T],
	logger *logan.Entry,
) *Consensus[T] {
	partiesMap := make(map[core.Address]p2p.Party, len(parties))
	for _, p := range parties {
		partiesMap[p.CoreAddress] = p
	}

	return &Consensus[T]{
		parties:     partiesMap,
		broadcaster: p2p.NewBroadcaster(parties),

		self:      party.Self,
		sessionId: party.SessionId,
		chainId:   party.ChainId,
		threshold: party.Threshold,

		db:          db,
		processor:   processor,
		constructor: constructor,

		logger: logger.WithField("session_id", party.SessionId),

		wg:   &sync.WaitGroup{},
		msgs: make(chan consensusMsg, msgsCapacity),
	}
}

type Consensus[T withdrawal.DepositSigningData] struct {
	parties     map[core.Address]p2p.Party
	broadcaster *p2p.Broadcaster

	self      core.Address
	sessionId string
	chainId   string
	threshold int

	db        db.DepositsQ
	processor *bridge.DepositFetcher

	constructor withdrawal.Constructor[T]

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
	case p2p.RequestType_RT_PROPOSAL, p2p.RequestType_RT_ACCEPTANCE, p2p.RequestType_RT_SIGN_START:
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
	if c.proposer == c.self {
		go c.propose(ctx)
	} else {
		go c.accept(ctx)
	}
}

func (c *Consensus[T]) WaitFor() (sigData *T, signers []p2p.Party, err error) {
	c.wg.Wait()
	c.ended.Store(true)

	return c.result.sigData, c.result.signers, c.result.err
}

func (c *Consensus[T]) determineProposer() core.Address {
	partyIds := make([]*tss.PartyID, len(c.parties)+1)
	idx := 0
	for _, party := range c.parties {
		partyIds[idx] = party.Identifier()
		idx++
	}
	partyIds[idx] = c.self.PartyIdentifier()

	sortedIds := tss.SortPartyIDs(partyIds)

	generator := deterministicRandSource(c.sessionId)
	proposerIdx := int(generator.Uint64() % uint64(sortedIds.Len()))

	return core.AddrFromPartyId(sortedIds[proposerIdx])
}

func deterministicRandSource(sessionId string) rand.Source {
	seed := sha256.Sum256([]byte(sessionId))
	return rand.NewChaCha8(seed)
}
