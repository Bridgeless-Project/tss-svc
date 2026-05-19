package tss

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
	"github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
	"github.com/taurusgroup/multi-party-sig/protocols/frost/sign"
)

type (
	Config        = keygen.Config
	TaprootConfig = keygen.TaprootConfig
	Signature     = sign.Signature
)

type KeygenParty struct {
	wg    *sync.WaitGroup
	ended atomic.Bool

	broadcaster  *broadcast.Broadcaster
	parties      map[core.Address]struct{}
	group        curve.Curve
	selfID       party.ID
	participants []party.ID
	threshold    int

	selfCoreAddress core.Address

	sessionId string
	handler   *protocol.MultiHandler

	msgs   chan tss.PartyMsg
	config *keygen.Config
	err    error
	logger *logan.Entry
}

func NewKeygenParty(self tss.LocalKeygenParty, group curve.Curve, parties []p2p.Party, threshold int, sessionId string, logger *logan.Entry) *KeygenParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyIds := make([]party.ID, 0, len(parties)+1)
	partyIds = append(partyIds, party.ID(self.Address.String()))

	for _, p := range parties {
		partyMap[p.CoreAddress] = struct{}{}
		partyIds = append(partyIds, party.ID(p.CoreAddress.String()))
	}
	participants := party.NewIDSlice(partyIds)

	return &KeygenParty{
		broadcaster:     broadcast.NewBroadcaster(parties, logger.WithField("component", "broadcaster")),
		parties:         partyMap,
		group:           group,
		selfID:          party.ID(self.Address.String()),
		participants:    participants,
		selfCoreAddress: self.Address,
		msgs:            make(chan tss.PartyMsg, tss.MsgsCapacity),

		threshold: threshold,
		logger:    logger,
		sessionId: sessionId,
		wg:        &sync.WaitGroup{},
	}
}

func (p *KeygenParty) Run(ctx context.Context) {

	fmt.Println("RUN FROST KEYGEN")
	h, err := protocol.NewMultiHandler(frost.Keygen(p.group, p.selfID, p.participants, p.threshold), []byte(p.sessionId))
	if err != nil {
		p.err = err
		p.ended.Store(false)
		p.logger.WithError(err).Error("failed to create frost keygen handler")
		return
	}
	p.handler = h

	p.wg.Add(2)
	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx)

	p.logger.Info("keygen started")
}

func (p *KeygenParty) WaitFor() *tss.LocalPartyData {
	fmt.Println("WAIT FOR FROST KEYGEN")
	p.wg.Wait()
	if p.err != nil || p.config == nil {
		p.logger.Error("keygen failed to wait for keygen")
		return nil
	}

	p.ended.Store(true)

	p.logger.Info("keygen finished")

	return tss.NewLocalPartyData(p.config)
}

func (p *KeygenParty) Receive(sender core.Address, data *p2p.TssData) {
	if p.ended.Load() {
		return
	}

	p.logger.Debug("Receive: received message", sender, data)

	p.msgs <- tss.PartyMsg{
		Sender:      sender,
		WireMsg:     data.Data,
		IsBroadcast: data.IsBroadcast,
	}
}

func (p *KeygenParty) receiveMsgs(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {

		case <-ctx.Done():
			p.logger.Warn("context is done; stopping receiving messages")
			return

		case msg, ok := <-p.msgs:
			if !ok {
				p.logger.Warn("channel closed; stopping receiving messages")
				return
			}
			p.logger.Info("received message", msg)

			if _, exists := p.parties[msg.Sender]; !exists {
				p.logger.WithField("party", core.Address(msg.Sender)).Warn("got message from outside party")
				continue
			}

			message := new(protocol.Message)
			if err := message.UnmarshalBinary(msg.WireMsg); err != nil {
				p.logger.WithError(err).WithField("party", core.Address(msg.Sender)).Warn("failed to unmarshal message")
				continue
			}

			p.logger.Info("received message", message)
			p.handler.Accept(message)
		}
	}
}

func (p *KeygenParty) receiveUpdates(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping listening to updates")
			return

		case msg, ok := <-p.handler.Listen():
			fmt.Println("Message to broadcast", msg)
			if !ok {
				r, err := p.handler.Result()
				if err != nil {
					p.err = err
					p.logger.WithError(err).Error("failed to get keygen result")
					return
				}

				if r == nil {
					p.err = errors.New("nil frost keygen result")
					p.logger.Error("failed to get keygen result")
					return
				}

				config, ok := r.(*Config)
				if !ok {
					p.err = errors.New("unexpected frost keygen result type")
					p.logger.WithField("type", r).Error("failed to get keygen result")
					return
				}
				p.config = config

				bytes, err := p.config.PublicKey.MarshalBinary()
				if err != nil {
					return
				}
				fmt.Println("\t\t\t\tp.config.PublicKey()\n:", hex.EncodeToString(bytes))
				p.ended.Store(true)
				close(p.msgs)
				return
			}

			p.logger.Debug("received update", msg)
			raw, err := msg.MarshalBinary()
			if err != nil {
				p.logger.WithError(err).Error("failed to marshal message")
				continue
			}

			tssData := &p2p.TssData{
				Data:        raw,
				IsBroadcast: msg.Broadcast,
			}

			tssReq, _ := anypb.New(tssData)
			submitReq := p2p.SubmitRequest{
				Sender:    p.selfCoreAddress.String(),
				SessionId: p.sessionId,
				Type:      p2p.RequestType_RT_KEYGEN,
				Data:      tssReq,
			}

			p.logger.Debug("sending request", submitReq)
			to := msg.To
			if to == "" {
				p.broadcaster.Broadcast(&submitReq)
				continue
			}

			p.logger.Debug("sending to", to)
			dst := core.AddrFromString(string(to))
			if err = p.broadcaster.Send(&submitReq, dst); err != nil {
				p.logger.WithError(err).Error("failed to send message")
			}
		}
	}
}
