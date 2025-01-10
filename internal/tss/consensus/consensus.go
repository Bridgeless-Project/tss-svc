package consensus

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
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
	sortedPartyIds tss.SortedPartyIDs
	parties        []p2p.Party
	partiesMap     map[core.Address]struct{}

	sessionId string

	rand         *rand.Rand
	formData     func([]byte) ([]byte, error)
	validateData func([]byte) (bool, error)

	resultData    []byte
	resultSigners []*tss.PartyID

	msgs  chan partyMsg
	ended atomic.Bool

	logger *logan.Entry
}

func (c *Consensus) Run(ctx context.Context) ([]byte, []p2p.Party, error) {

	// 1. Pick a proposer for this Consensus session
	id := c.rand.IntN(c.sortedPartyIds.Len())
	c.proposerID = tss.NewPartyID(strconv.Itoa(id), "proposer", big.NewInt(int64(id)))

	if id == c.self.Address.PartyIdentifier().Index {
		c.self.partyStatus = Proposer
		c.logger.Info("successfully picked a proposer, proposer id is ", id)
	}
	c.wg.Add(1)
	// 2.1 If local party is proposer - validate incoming data and form data to sign and send it to signers
	if c.self.partyStatus == Proposer {
		//TODO: Add data parsing and validation logic
		//TODO: Add data is not null validation with broadcasting NO_DATA_TO_SIGN message type
		var inputData []byte //input data as example

		if inputData == nil {
			c.sendMessage(inputData, nil, p2p.RequestType_NO_DATA_TO_SIGN)
			c.waitFor()
			return nil, nil, errors.Wrap(errors.New("nil data"), "no input data")
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
		go c.receiveMsgs(ctx)

	}
	// 2.2 If party status is signer in only receives messages

	if c.self.partyStatus == Signer {
		go c.receiveMsgs(ctx)
	}

	// 3 Validate threshold
	if c.params.Threshold < c.sortedPartyIds.Len() {
		return nil, nil, errors.Wrap(errors.New("consensus failure"), "not enough signers fro threshold")
	}

	signers := c.parties[:len(c.resultSigners)]
	return c.resultData, signers, nil
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
			dst := core.AddrFromPartyId(dst.Identifier())
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

func (c *Consensus) waitFor() {
	c.wg.Wait()
	c.ended.Store(true)
}

func (c *Consensus) receiveMsgs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Warn("context is done; stopping receiving messages")
			return
		case msg, ok := <-c.msgs:
			if !ok {
				c.logger.Debug("msg channel is closed")
				return
			}

			if _, exists := c.partiesMap[msg.Sender]; !exists {
				c.logger.WithField("party", msg.Sender).Warn("got message from outside party")
				continue
			}

			if c.self.partyStatus == Proposer {
				if msg.Type == p2p.RequestType_ACK {
					c.resultSigners = append(c.resultSigners, msg.Sender.PartyIdentifier())
				}
			}
			if c.self.partyStatus == Signer || c.self.partyStatus == Proposer {
				if msg.Type == p2p.RequestType_DATA_TO_SIGN {
					//perform validation by signer
					valid, err := c.validateData(msg.WireMsg)
					if err != nil {
						c.logger.Error("failed to validate data", err)
						c.waitFor()
					}
					if !valid {
						c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
						c.waitFor()
					}
					if valid {
						c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_ACK)
						c.waitFor()
					}
				}
			}
		}
	}
}

func (c *Consensus) Id() string {
	return c.sessionId
}
func (c *Consensus) Receive(request *p2p.SubmitRequest) error {
	if request == nil || request.Data == nil {
		return errors.New("nil request")
	}
	if request.Type != p2p.RequestType_ACK || request.Type != p2p.RequestType_NACK || request.Type != p2p.RequestType_DATA_TO_SIGN || request.Type != p2p.RequestType_NO_DATA_TO_SIGN {
		return errors.New("invalid request type")
	}

	data := &p2p.TssData{}
	if err := request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal TSS request data")
	}

	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}

	if c.ended.Load() {
		return errors.Wrap(errors.New("consensus ended already"), "")
	}

	c.msgs <- partyMsg{
		Type:        request.Type,
		Sender:      sender,
		WireMsg:     data.Data,
		IsBroadcast: data.IsBroadcast,
	}
	return nil
}

// RegisterIdChangeListener is a no-op for Consensus
func (c *Consensus) RegisterIdChangeListener(func(oldId string, newId string)) {
}
