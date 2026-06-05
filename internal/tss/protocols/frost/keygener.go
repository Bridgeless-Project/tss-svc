package tss

import (
	"context"
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
)

type KeygenParty struct {
	wg    *sync.WaitGroup
	ended atomic.Bool

	broadcaster *broadcast.Broadcaster
	parties     map[core.Address]struct{}
	group       curve.Curve

	self         tss.LocalKeygenParty
	participants []party.ID

	sessionId string
	handler   *protocol.MultiHandler

	msgs   chan tss.PartyMsg
	once   sync.Once
	result *FrostShare

	err    error
	logger *logan.Entry
}

func NewKeygenParty(self tss.LocalKeygenParty, group curve.Curve, parties []p2p.Party, sessionId string, logger *logan.Entry) *KeygenParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyIds := make([]party.ID, 0, len(parties)+1)
	partyIds = append(partyIds, party.ID(self.Address.String()))

	for _, p := range parties {
		partyMap[p.CoreAddress] = struct{}{}
		partyIds = append(partyIds, party.ID(p.CoreAddress.String()))
	}
	participants := party.NewIDSlice(partyIds)

	return &KeygenParty{
		self:         self,
		broadcaster:  broadcast.NewBroadcaster(parties, logger.WithField("component", "broadcaster")),
		parties:      partyMap,
		group:        group,
		participants: participants,
		msgs:         make(chan tss.PartyMsg, tss.MsgsCapacity),
		result:       NewFrostShare(),

		logger:    logger.WithField("protocol", "frost"),
		sessionId: sessionId,
		wg:        new(sync.WaitGroup),
	}
}

func (p *KeygenParty) Run(ctx context.Context) {
	h, err := protocol.NewMultiHandler(frost.Keygen(p.group, party.ID(p.self.Address.String()), p.participants, p.self.Threshold), []byte(p.sessionId))
	if err != nil {
		p.err = err
		p.finish()
		p.logger.WithError(err).Error("failed to create frost keygen handler")
		return
	}
	p.handler = h

	p.wg.Add(2)
	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx)

	p.logger.Info("keygen started")
}

func (p *KeygenParty) WaitFor() tss.Share {
	p.wg.Wait()
	if p.err != nil || p.result == nil {
		p.logger.Error("keygen failed to wait for keygen")
		return nil
	}

	p.ended.Store(true)

	p.logger.Info("keygen finished")

	return p.result
}

func (p *KeygenParty) Receive(sender core.Address, data *p2p.TssData) {
	if p.ended.Load() {
		return
	}

	p.logger.Debug("received message", sender, data)

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
				p.logger.WithField("party", msg.Sender).Warn("got message from outside party")
				continue
			}

			message := new(protocol.Message)
			if err := message.UnmarshalBinary(msg.WireMsg); err != nil {
				p.logger.WithError(err).WithField("party", msg.Sender).Warn("failed to unmarshal message")
				continue
			}
			if err := validateMessageEnvelope(msg.Sender, party.ID(p.self.Address.String()), msg, message); err != nil {
				p.logger.WithError(err).WithField("party", msg.Sender).Warn("rejected invalid frost message envelope")
				continue
			}

			p.logger.Info("received message", message)
			p.handler.Accept(message)
		}
	}
}

func (p *KeygenParty) receiveUpdates(ctx context.Context) {
	defer func() {
		p.finish()
		p.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping listening to updates")
			return

		case msg, ok := <-p.handler.Listen():
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

				config, ok := r.(*frost.Config)
				if !ok {
					p.err = errors.New("unexpected frost keygen result type")
					p.logger.WithField("type", r).Error("failed to get keygen result")
					return
				}
				pk, _ := config.PublicKey.MarshalBinary()
				fmt.Println("tp.result.PubKey():", pk)

				err = p.result.SetData(config)
				if err != nil {
					p.err = err
					p.logger.WithError(err).Error("failed to set keygen result")
					return
				}
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
				Sender:    p.self.Address.String(),
				SessionId: p.sessionId,
				Type:      p2p.RequestType_RT_KEYGEN,
				Data:      tssReq,
			}

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

func (p *KeygenParty) finish() {
	p.once.Do(func() {
		p.ended.Store(true)
		close(p.msgs)
	})
}
