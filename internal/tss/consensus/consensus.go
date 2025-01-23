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

// FIXME: CHANGE TO DEPOSIT DATA
type DepositSigningData interface {
	Deposit() Deposit
	ToPayload() *anypb.Any
	FromPayload(payload *anypb.Any) error
}

// FIXME: CHANGE TO DEPOSIT DATA
type SigDataFormer[T DepositSigningData] interface {
	FormSigningData(Deposit) (T, error)
}

type SigDataValidator[T DepositSigningData] interface {
	IsValid(data T) (bool, error)
}

type Deposit struct {
}

type LocalConsensusParty struct {
	Self      core.Address
	SessionId string
	Threshold int
}

type Consensus[T DepositSigningData] struct {
	parties     map[core.Address]p2p.Party
	broadcaster *p2p.Broadcaster

	self      core.Address
	sessionId string
	threshold int

	sigDataFormer    SigDataFormer[T]
	sigDataValidator SigDataValidator[T]

	logger *logan.Entry

	proposer core.Address
	wg       *sync.WaitGroup
	ended    atomic.Bool
	msgs     chan consensusMsg

	result struct {
		sigData T
		signers []p2p.Party
		err     error
	}
}

func (c *Consensus[T]) Receive(request *p2p.SubmitRequest) error {
	if request == nil || request.Data == nil {
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
	// todo: implement
	// todo: check not nil

	c.wg.Add(1)
	if c.proposer == c.self {
		go c.propose(ctx)
	} else {
		go c.accept(ctx)
	}
}

func (c *Consensus[T]) WaitFor() (sigData T, signers []p2p.Party, err error) {
	c.wg.Wait()
	c.ended.Store(true)

	return c.result.sigData, c.result.signers, c.result.err
}

func (c *Consensus[T]) determineProposer() core.Address {
	partyIds := make([]*tss.PartyID, 0, len(c.parties)+1)
	partyIds[0] = c.self.PartyIdentifier()
	for _, party := range c.parties {
		partyIds = append(partyIds, party.Identifier())
	}
	sortedIds := tss.SortPartyIDs(partyIds)

	generator := deterministicRandSource(c.sessionId)
	proposerIdx := int(generator.Uint64() % uint64(sortedIds.Len()))

	return core.AddrFromPartyId(partyIds[proposerIdx])
}

func deterministicRandSource(sessionId string) rand.Source {
	seed := sha256.Sum256([]byte(sessionId))
	return rand.NewChaCha8(seed)
}
