package consensus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
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

	chainData    chain.ChainMetadata
	rand         *rand.Rand
	data         []byte
	formData     func([]byte) ([]byte, error)
	validateData func([]byte) (bool, error)
	dataSelector func(string, []byte) ([]byte, error)

	resultData    []byte
	resultSigners []p2p.Party
	err           error

	msgs    chan partyMsg
	ended   atomic.Bool
	chainId string

	logger *logan.Entry
}

func NewConsensus(self LocalParams, parties []p2p.Party, logger *logan.Entry, sessionId string, data []byte, formData func([]byte) ([]byte, error), validateData func([]byte) (bool, error), threshold int, metadata chain.ChainMetadata, chainId string, dataSelector func(string, []byte) ([]byte, error)) *Consensus {
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
		wg:             &sync.WaitGroup{},
		self:           self,
		sessionId:      sessionId,
		broadcaster:    p2p.NewBroadcaster(parties),
		chainData:      metadata,
		dataSelector:   dataSelector,
		chainId:        chainId,
		parties:        parties,
		partiesMap:     partyMap,
		ackSet:         make(map[string]struct{}),
		data:           data,
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
	// 2.1 If local party is proposer - validate incoming data and form data to sign and send it to signers
	if c.self.PartyStatus == Proposer {
		// select data with selector
		c.data, c.err = c.dataSelector(c.chainId, c.data)
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
		dst := core.AddrFromPartyId(to)
		if err := c.broadcaster.Send(&submitReq, dst); err != nil {
			c.logger.WithError(err).Error("failed to send message")
		}
	}
	if messageType == p2p.RequestType_SIGNER_NOTIFY {
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
	defer c.wg.Done()
	votesCount := 0
	for {
		select {
		case <-ctx.Done():
			c.logger.Warnf("context timed out with %d ACKs out of %d needed", len(c.ackSet), c.threshold)
			if len(c.ackSet) < c.threshold {
				c.logger.Error("Consensus failed due to insufficient ACKs")
			}
			close(c.msgs)
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
					if _, exists := c.ackSet[msg.Sender.PartyKey().String()]; !exists {
						c.ackSet[msg.Sender.PartyKey().String()] = struct{}{}
						votesCount++
						c.logger.Info("Received ACK from party", msg.Sender.PartyIdentifier())

						if votesCount == len(c.parties) {
							c.logger.Info("All parties voted")
							if len(c.ackSet) < c.threshold {
								c.logger.Error("Didn`t reach threshold")
								c.err = errors.Wrap(errors.New("consensus failed"), "didn`t reach threshold")
							}
							var signerKeysList []string
							for signerKey, _ := range c.ackSet {
								for _, party := range c.parties {
									if party.CoreAddress.PartyKey().String() == signerKey {
										c.resultSigners = append(c.resultSigners, party)
										break
									}
								}
								signerKeysList = append(signerKeysList, signerKey)
							}
							c.resultSigners = c.resultSigners[:c.threshold]
							c.logger.Info("Signers list", c.resultSigners)

							err := c.notifySigners(signerKeysList)
							if err != nil {
								c.logger.Error("failed to notify signers", err)
								c.err = errors.Wrap(err, "failed to notify signers")
							}
							close(c.msgs)
							return
						}
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

			if msg.Type == p2p.RequestType_DATA_TO_SIGN {
				//perform validation by signer

				//validate sender
				senderId := msg.Sender.PartyIdentifier().Id
				if senderId != c.proposerKey {
					c.logger.Error("invalid proposer")
					c.err = errors.New("invalid proposer")
					c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
					close(c.msgs)
					return
				}
				// validate deposit data with recreating it
				localData, err := c.formData(c.data)
				if err != nil {
					c.logger.Error("failed to form data", err)
					c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
					close(c.msgs)
					return
				}
				if !bytes.Equal(localData, msg.WireMsg) {
					c.logger.Error("invalid data")
					c.err = errors.Wrap(errors.New("invalid data"), "formed different data")
					c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
				}
				c.logger.Info("got new data: ", msg.WireMsg)
				c.resultData = msg.WireMsg
				c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_ACK)
				//
				//valid, err := c.validateData(msg.WireMsg)
				//if err != nil {
				//	c.logger.Error("failed to validate data ", err)
				//	c.err = errors.Wrap(err, "failed to validate data")
				//	c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
				//	close(c.msgs)
				//	return
				//
				//}
				//if !valid {
				//	c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
				//	close(c.msgs)
				//	return
				//}
				//if valid {
				//}
				continue
			}
			if msg.Type == p2p.RequestType_NO_DATA_TO_SIGN {
				close(c.msgs)
				c.resultData = nil
				c.resultSigners = nil
				c.err = errors.New(string(msg.WireMsg))
				c.logger.Warn("got no data")
				return
			}

			if msg.Type == p2p.RequestType_SIGNER_NOTIFY {
				var signersList []string
				err := json.Unmarshal(msg.WireMsg, &signersList)
				if err != nil {
					c.logger.Error("failed to unmarshal signer list", err)
				}
				for _, signer := range signersList {
					for _, party := range c.parties {
						if party.CoreAddress.PartyKey().String() == signer {
							c.resultSigners = append(c.resultSigners, party)
						}
					}
				}
				c.logger.Info("Signer list received", c.resultSigners)
				close(c.msgs)
				return
			}
		}
	}
}

func (c *Consensus) notifySigners(signers []string) error {
	resultSignersData, err := json.Marshal(signers)
	if err != nil {
		return errors.Wrap(err, "failed to serialize resultSigners")
	}

	for _, signer := range c.resultSigners {
		c.sendMessage(resultSignersData, signer.CoreAddress.PartyIdentifier(), p2p.RequestType_SIGNER_NOTIFY)
	}
	return nil
}
