package consensus

import (
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	tss2 "github.com/hyle-team/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"math/big"
	"math/rand/v2"
	"strconv"
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
	partyStatus partyStatus //init as Signer
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

	proposerID     *tss.PartyID
	broadcaster    *p2p.Broadcaster
	party          p2p.Party
	sortedPartyIds tss.SortedPartyIDs
	parties        []*tss.PartyID

	sessionId string

	rand         *rand.Rand
	formData     func([]byte) ([]byte, error)
	validateData func([]byte) (bool, error)

	resultData    []byte
	resultSigners []tss.PartyID

	msgs  chan tss2.PartyMsg
	ended atomic.Bool

	logger *logan.Entry
}

func (c *Consensus) Run() ([]byte, []tss.PartyID, error) {

	// 1. Pick a proposer for this Consensus session
	id := c.rand.IntN(c.sortedPartyIds.Len())
	c.proposerID = tss.NewPartyID(strconv.Itoa(id), "proposer", big.NewInt(int64(id)))

	if id == c.self.Address.PartyIdentifier().Index {
		c.self.partyStatus = Proposer
		c.logger.Info("successfully picked a proposer, proposer id is ", id)
	}

	// 2.1 If local party is proposer - validate incoming data and form data to sign and send it to signers
	if c.self.partyStatus == Proposer {
		//TODO: Add data parsing and validation logic
		//TODO: Add data is not null validation with broadcasting NO_DATA_TO_SIGN message type
		var inputData []byte //input data as example

		if inputData == nil {
			c.sendMessage(inputData, nil, p2p.RequestType_NO_DATA_TO_SIGN)
			return nil, []tss.PartyID{}, nil
		}

		valid, err := c.validateData(inputData)
		if err != nil {
			c.logger.Error("failed to validate input data", err)
			return nil, nil, errors.Wrap(err, "failed to validate input data")
		}
		if !valid {
			return nil, nil, errors.Wrap(errors.New("invalid data"), "invalid input data")
		}

		c.resultData, err = c.formData(inputData) //will be returned after successful consensus process
		if err != nil {
			c.logger.Error("failed to form data", err)
			return nil, nil, errors.Wrap(err, "failed to form data")
		}
		// Send data to
		c.sendMessage(c.resultData, nil, p2p.RequestType_DATA_TO_SIGN)
		//TODO: add receiving ACK and NACK
	}
	// 2.2 If party status is signer in only receives messages

	if c.self.partyStatus == Signer {
		//TODO: add receiveng messages from proposer
	}

	// 3 Validate threshold
	if c.params.Threshold < c.sortedPartyIds.Len() {

	}
	return c.resultData, c.resultSigners, nil
}

// sendMessage is general func to send messages during consensus process
func (c *Consensus) sendMessage(data []byte, to *tss.PartyID, messageType p2p.RequestType) {
	if messageType == p2p.RequestType_DATA_TO_SIGN || messageType == p2p.RequestType_NO_DATA_TO_SIGN {

		tssData := &p2p.TssData{
			Data:        data,
			IsBroadcast: true,
		}

		tssReq, _ := anypb.New(tssData)
		submitReq := p2p.SubmitRequest{
			Sender:    c.self.Address.String(),
			SessionId: c.sessionId,
			Type:      messageType,
			Data:      tssReq,
		}

		for _, dst := range c.parties {
			dst := core.AddrFromPartyId(dst)
			if err := c.broadcaster.Send(&submitReq, dst); err != nil {
				c.logger.WithError(err).Error("failed to send message")
			}
		}
	}
	if messageType == p2p.RequestType_ACK || messageType == p2p.RequestType_NACK {
	}
	tssData := &p2p.TssData{
		Data:        data,
		IsBroadcast: false,
	}

	tssReq, _ := anypb.New(tssData)
	submitReq := p2p.SubmitRequest{
		Sender:    c.self.Address.String(),
		SessionId: c.sessionId,
		Type:      messageType,
		Data:      tssReq,
	}
	dst := core.AddrFromPartyId(to)
	if err := c.broadcaster.Send(&submitReq, dst); err != nil {
		c.logger.WithError(err).Error("failed to send message")
	}
}
