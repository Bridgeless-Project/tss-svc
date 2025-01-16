package consensus

import (
	"context"
	"crypto/sha256"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	tss2 "github.com/hyle-team/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

type PartyStatus int

const (
	Proposer PartyStatus = iota
	Signer
)

type LocalParams struct {
	PartyStatus PartyStatus //init as Signer
	Address     core.Address
}

type Consensus struct {
	wg        *sync.WaitGroup
	self      LocalParams
	sessionId string

	broadcaster    *p2p.Broadcaster
	parties        []p2p.Party
	sortedPartyIds []*tss.PartyID
	partiesMap     map[core.Address]struct{}
	ackSet         map[string]struct{}
	threshold      int
	proposerKey    string

	chainData chain.Chain
	chainId   string
	//chainClient bridgeTypes.Proxy

	testData     []byte // mock will be deleted
	rand         *rand.Rand
	data         []byte
	formData     func([]byte) ([]byte, error)
	validateData func([]byte) (bool, error)
	dataSelector func(string, []byte) ([]byte, error)

	resultData    []byte
	resultSigners []p2p.Party
	err           error

	msgs  chan partyMsg
	ended atomic.Bool

	logger *logan.Entry
}

func NewConsensus(self LocalParams, parties []p2p.Party, logger *logan.Entry, sessionId string, data []byte, formData func([]byte) ([]byte, error), validateData func([]byte) (bool, error), threshold int, metadata chain.Chain, chainId string, dataSelector func(string, []byte) ([]byte, error)) *Consensus {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyMap[self.Address] = struct{}{}
	partyIds := make([]*tss.PartyID, len(parties)+1)
	partyIds[0] = self.Address.PartyIdentifier()

	for i, party := range parties {
		if party.CoreAddress == self.Address {
			continue
		}

		partyMap[party.CoreAddress] = struct{}{}
		partyIds[i+1] = party.Identifier()
	}

	return &Consensus{
		wg:           &sync.WaitGroup{},
		self:         self,
		sessionId:    sessionId,
		broadcaster:  p2p.NewBroadcaster(parties),
		chainData:    metadata,
		dataSelector: dataSelector,
		chainId:      chainId,
		// chainClient:    proxy,
		parties:        parties,
		partiesMap:     partyMap,
		ackSet:         make(map[string]struct{}),
		testData:       data,
		formData:       formData,
		sortedPartyIds: tss.SortPartyIDs(partyIds),
		validateData:   validateData,
		msgs:           make(chan partyMsg, tss2.MsgsCapacity),
		ended:          atomic.Bool{},
		logger:         logger,
		threshold:      threshold,
	}
}

func (c *Consensus) Run(ctx context.Context) {

	// 1. Pick a proposer for this Consensus session
	var seed [32]byte
	hash := sha256.Sum256([]byte(c.sessionId))
	copy(seed[:], hash[:])
	gen := rand.NewChaCha8(seed)
	randIndex := int(gen.Uint64() % uint64(len(c.parties)))
	c.proposerKey = c.sortedPartyIds[randIndex].Id

	c.logger.Info("proposer ID ", c.proposerKey)
	c.logger.Info("local key", c.self.Address.PartyIdentifier().Id)

	if c.proposerKey == c.self.Address.PartyIdentifier().Id {
		c.logger.Info("i am a proposer")
		c.self.PartyStatus = Proposer
	}
	c.wg.Add(1)
	c.logger.Info("data on start: ", c.data)
	// 2.1 If local party is proposer - validate incoming data and form data to sign and send it to signers
	if c.self.PartyStatus == Proposer {
		// select data with selector
		c.data, c.err = c.dataSelector(c.chainId, c.testData)
		if c.err != nil {
			c.resultData = nil
			c.sendMessage([]byte(c.err.Error()), nil, p2p.RequestType_NO_DATA_TO_SIGN)
			return
		}
		if c.data == nil {
			c.err = errors.Wrap(errors.New("nil data"), "no input data")
			c.sendMessage([]byte(c.err.Error()), nil, p2p.RequestType_NO_DATA_TO_SIGN)
			return
		}
		valid, err := c.validateData(c.data)
		if err != nil {
			c.sendMessage([]byte(err.Error()), nil, p2p.RequestType_NO_DATA_TO_SIGN)
			err = errors.Wrap(err, "failed to validate input data")
			return
		}
		if !valid {
			c.err = errors.New("invalid data")
			c.sendMessage([]byte(c.err.Error()), nil, p2p.RequestType_NO_DATA_TO_SIGN)
			return
		}
		// share signers picked raw transfer data
		c.sendMessage(c.data, nil, p2p.RequestType_RAW_DATA)
		c.resultData, err = c.formData(c.data) //will be returned after successful consensus process
		if err != nil {
			c.err = errors.Wrap(err, "failed to form data")
			c.sendMessage([]byte(c.err.Error()), nil, p2p.RequestType_NO_DATA_TO_SIGN)
			return
		}
		// Send data to parties
		c.ackSet[c.self.Address.PartyKey().String()] = struct{}{}
		c.sendMessage(c.resultData, nil, p2p.RequestType_DATA_TO_SIGN)
		go c.receiveMsgs(ctx)

	}
	if c.self.PartyStatus == Signer {
		go c.receiveMsgs(ctx)
	}
	return
}

// sendMessage is general func to send messages during consensus process
func (c *Consensus) sendMessage(data []byte, to *tss.PartyID, messageType p2p.RequestType) {
	c.logger.Info("message type ", messageType.String())
	submitReq := p2p.SubmitRequest{
		Sender:    c.self.Address.String(),
		SessionId: c.sessionId,
		Type:      messageType,
		Data:      &anypb.Any{},
	}
	tssData := &p2p.TssData{
		Data:        data,
		IsBroadcast: true,
	}
	if messageType == p2p.RequestType_DATA_TO_SIGN || messageType == p2p.RequestType_NO_DATA_TO_SIGN || messageType == p2p.RequestType_RAW_DATA {
		tssReq, _ := anypb.New(tssData)
		submitReq.Data = tssReq
		for _, dst := range c.parties {
			dst := core.AddrFromPartyId(dst.Identifier())
			if err := c.broadcaster.Send(&submitReq, dst); err != nil {
				c.logger.WithError(err).Error("failed to send message")
			}
		}
	}
	if messageType == p2p.RequestType_ACK || messageType == p2p.RequestType_NACK {
		tssData.IsBroadcast = true
		tssReq, _ := anypb.New(tssData)
		submitReq.Data = tssReq
		dst := core.AddrFromPartyId(to)
		if err := c.broadcaster.Send(&submitReq, dst); err != nil {
			c.logger.WithError(err).Error("failed to send message")
		}
	}
	if messageType == p2p.RequestType_SIGNER_NOTIFY {
		tssReq, _ := anypb.New(tssData)
		submitReq.Data = tssReq
		dst := core.AddrFromPartyId(to)
		if err := c.broadcaster.Send(&submitReq, dst); err != nil {
			c.logger.WithError(err).Error("failed to send message")
		}
	}
}

func (c *Consensus) WaitFor() ([]byte, []p2p.Party, error) {
	c.wg.Wait()
	c.ended.Store(true)

	// If party is not a signer it won`t receive the list of signers
	if c.resultSigners == nil {
		c.resultData = nil
	}

	if len(c.resultSigners) != c.threshold {
		c.err = errors.Wrap(errors.New("consensus failed"), "didn`t reached threshold")
	}

	return c.resultData, c.resultSigners, c.err
}

func (c *Consensus) Receive(sender core.Address, data *p2p.TssData, reqType p2p.RequestType) {
	if c.ended.Load() {
		return
	}

	c.msgs <- partyMsg{
		Type:        reqType,
		Sender:      sender,
		WireMsg:     data.Data,
		IsBroadcast: data.IsBroadcast,
	}

}
func (c *Consensus) receiveMsgs(ctx context.Context) {
	defer func() {
		c.wg.Done()
		close(c.msgs)
	}()
	votesCount := 0
	for {
		select {
		case <-ctx.Done():
			c.logger.Warnf("context timed out with %d ACKs out of %d needed", len(c.ackSet), c.threshold)
			if len(c.ackSet) < c.threshold {
				c.logger.Error("Consensus failed due to insufficient ACKs")
			}
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

			if c.self.PartyStatus == Proposer {
				if msg.Type == p2p.RequestType_ACK {
					c.handleACK(msg, &votesCount)
					if votesCount == len(c.parties) {
						c.finalizeConsensus()
						return
					}
					continue
				}
				if msg.Type == p2p.RequestType_NACK {
					if _, exists := c.ackSet[msg.Sender.PartyKey().String()]; !exists {
						votesCount++
						c.logger.Info("Received NACK from party", msg.Sender.PartyIdentifier())
					}
					continue
				}
			}
			// perform data validation
			if msg.Type == p2p.RequestType_DATA_TO_SIGN {
				err := c.validateIncomingData(msg)
				if err != nil {
					c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
				}
				c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_ACK)
				continue
			}
			// end the consensus with error
			if msg.Type == p2p.RequestType_NO_DATA_TO_SIGN {
				c.resultData = nil
				c.resultSigners = nil
				c.err = errors.New(string(msg.WireMsg))
				c.logger.Warn("got no data")
				return
			}

			if msg.Type == p2p.RequestType_SIGNER_NOTIFY {
				c.receiveSignerNotification(msg)
				return
			}
			if msg.Type == p2p.RequestType_RAW_DATA {
				// maybe do a validation by signer???
				senderId := msg.Sender.PartyIdentifier().Id
				if senderId != c.proposerKey {
					c.logger.Error("invalid proposer")
					c.err = errors.New("invalid proposer")
					return
				}
				c.data = msg.WireMsg
				c.logger.Info("got data ", c.data)
				continue
			}
		}
	}
}
