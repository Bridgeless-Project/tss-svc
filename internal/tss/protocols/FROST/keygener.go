package tss

import (
	"context"

	"sync"
	"sync/atomic"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	tss2 "github.com/Bridgeless-Project/tss-svc/internal/tss"
	bnb "github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
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

	config *keygen.Config
	logger *logan.Entry
}

func NewKeygenParty(self tss2.LocalKeygenParty, parties []p2p.Party, threshold int, sessionId string, logger *logan.Entry) *KeygenParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyIds := make([]party.ID, len(parties)+1)
	partyIds[0] = party.ID(self.Address.PartyIdentifier().Id)

	for i, p := range parties {
		partyMap[p.CoreAddress] = struct{}{}
		partyIds[i+1] = party.ID(p.Identifier().Id)
	}

	return &KeygenParty{
		broadcaster: broadcast.NewBroadcaster(parties, logger.WithField("component", "broadcaster")),
		parties:     partyMap,

		threshold: threshold,
		logger:    logger,
		sessionId: sessionId,
		wg:        &sync.WaitGroup{},
	}
}

func (p *KeygenParty) Run(ctx context.Context) {

	p.wg.Add(2)

	h, err := protocol.NewMultiHandler(frost.Keygen(p.group, p.selfID, p.participants, p.threshold), []byte(p.sessionId))
	if err != nil {
		return
	}
	p.handler = h

	out := make(chan bnb.Message, tss2.OutChannelSize)

	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx, out)

	p.logger.Info("keygen started")
}

func (p *KeygenParty) WaitFor() *tss.LocalPartyData {
	p.wg.Wait()
	p.ended.Store(true)

	p.logger.Info("keygen finished")

	return nil
}

func (p *KeygenParty) Receive(sender core.Address, data *p2p.TssData) {
	//	if p.ended.Load() {
	//		return
	//	}
	//
	//	p.msgs <- partyMsg{
	//		Sender:      sender,
	//		WireMsg:     data.Data,
	//		IsBroadcast: data.IsBroadcast,
	//	}
}

func (p *KeygenParty) receiveMsgs(ctx context.Context) {
	for {
		select {

		case <-ctx.Done():
			return

		// outgoing messages
		case msg, ok := <-p.handler.Listen():
			if !ok {
				r, err := p.handler.Result()
				if err != nil {
					p.logger.Error("failed to get keygen result")
					return
				}

				if r == nil {
					p.logger.Error("failed to get keygen result")
					return
				}

				p.config = r.(*Config)

			}

			if _, exists := p.parties[core.Address(msg.From)]; !exists {
				p.logger.WithField("party", core.Address(msg.From)).Warn("got message from outside party")
				continue
			}

			p.handler.Accept(msg)
		}
	}
}

func (p *KeygenParty) receiveUpdates(ctx context.Context, out <-chan bnb.Message) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping listening to updates")
			return

		//return
		case msg := <-out:
			raw, routing, err := msg.WireBytes()
			if err != nil {
				p.logger.WithError(err).Error("failed to get message wire bytes")
				continue
			}

			tssData := &p2p.TssData{
				Data:        raw,
				IsBroadcast: routing.IsBroadcast,
			}

			tssReq, _ := anypb.New(tssData)
			submitReq := p2p.SubmitRequest{
				Sender:    p.selfCoreAddress.String(),
				SessionId: p.sessionId,
				Type:      p2p.RequestType_RT_KEYGEN,
				Data:      tssReq,
			}

			// https://github.com/bnb-chain/tss/blob/100c015447e557b0608c8c8cbd30730d5dac7fba/client/client.go#L288
			to := routing.To
			if to == nil || len(to) > 1 {
				p.broadcaster.Broadcast(&submitReq)
				continue
			}

			dst := core.AddrFromPartyId(to[0])
			if err = p.broadcaster.Send(&submitReq, dst); err != nil {
				p.logger.WithError(err).Error("failed to send message")
			}
		}
	}
}
