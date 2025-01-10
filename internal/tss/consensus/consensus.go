package consensus

import (
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	tss2 "github.com/hyle-team/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

type partyStatus int

const (
	Proposer partyStatus = iota
	Signer
)

type LocalParams struct {
	partyStatus partyStatus
	Address     core.Address
}

type ConsensusParams struct {
	StartTime time.Time
	Threshold int
}

type Consensus struct {
	wg     *sync.WaitGroup
	params ConsensusParams
	self   LocalParams

	proposerID     tss.PartyID
	broadcaster    *p2p.Broadcaster
	party          p2p.Party
	sortedPartyIds tss.SortedPartyIDs
	parties        map[core.Address]struct{}

	sessionId string

	rand         *rand.Rand
	formData     func([]byte) ([]byte, error)
	validateData func([]byte) (bool, error)

	msgs  chan tss2.PartyMsg
	ended atomic.Bool

	logger *logan.Entry
}

func (c *Consensus) Run() ([]byte, []tss.PartyID, error) {
	// 1. Pick a proposer for this Consensus session
	id := c.rand.IntN(c.sortedPartyIds.Len())
	if id == c.self.Address.PartyIdentifier().Index {
		c.self.partyStatus = Proposer
		c.logger.Info("successfully picked a proposer, proposer id is ", id)
	}

	// 2.1 If local party is proposer - validate incoming data and form data to sign and send it to signers

	//TODO: Add data parsing and validation logic
	var inputData []byte //input data as example

	valid, err := c.validateData(inputData)
	if err != nil {
		c.logger.Error("failed to validate input data", err)
		return nil, nil, errors.Wrap(err, "failed to validate input data")
	}
	if !valid {
		return nil, nil, errors.Wrap(errors.New("invalid data"), "invalid input data")
	}

	dataToSign, err := c.formData(inputData) //will be returned after successful consensus process
	if err != nil {
		c.logger.Error("failed to form data", err)
		return nil, nil, errors.Wrap(err, "failed to form data")
	}

	// 2.2

}
