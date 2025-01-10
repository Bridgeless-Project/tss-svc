package tss

import (
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"gitlab.com/distributed_lab/logan/v3"
	"sync"
	"sync/atomic"
)

type PartyStatus int

const (
	Proposer PartyStatus = iota
	Signer   PartyStatus = iota
)

type LocalConsensusParams struct {
	signingSessionParams session.SigningSessionParams
	localSignParams      LocalSignParty
	partyStatus          PartyStatus
}

type ConsensusParty struct {
	wg *sync.WaitGroup

	parties        map[core.Address]struct{}
	sortedPartyIds tss.SortedPartyIDs

	self LocalConsensusParams

	logger      *logan.Entry
	party       tss.Party
	msgs        chan partyMsg
	broadcaster *p2p.Broadcaster

	data []byte

	// maybe add something like chain Params to pick tx`s for specific chain

	ended      atomic.Bool
	result     *common.SignatureData
	sessionId  string
	proposerId *tss.PartyID

	formDataFunc     func([]byte) []byte
	validateDataFunc func([]byte) bool
}
