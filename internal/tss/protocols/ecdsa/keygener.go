package tss

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	bnb "github.com/bnb-chain/tss-lib/v2/tss"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

type KeygenParty struct {
	wg    *sync.WaitGroup
	ended atomic.Bool

	broadcaster    *broadcast.Broadcaster
	party          bnb.Party
	sortedPartyIds bnb.SortedPartyIDs
	parties        map[core.Address]struct{}
	self           tss.LocalKeygenParty

	msgs      chan tss.PartyMsg
	result    *keygen.LocalPartySaveData
	sessionId string

	logger *logan.Entry
}

func NewKeygenParty(self tss.LocalKeygenParty, parties []p2p.Party, sessionId string, logger *logan.Entry) *KeygenParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyIds := make([]*bnb.PartyID, len(parties)+1)
	partyIds[0] = self.Address.PartyIdentifier()

	for i, party := range parties {
		partyMap[party.CoreAddress] = struct{}{}
		partyIds[i+1] = party.Identifier()
	}

	return &KeygenParty{
		broadcaster:    broadcast.NewBroadcaster(parties, logger.WithField("component", "broadcaster")),
		sortedPartyIds: bnb.SortPartyIDs(partyIds),
		parties:        partyMap,
		self:           self,
		msgs:           make(chan tss.PartyMsg, tss.MsgsCapacity),
		logger:         logger.WithField("protocol", "ecdsa"),
		sessionId:      sessionId,
		wg:             &sync.WaitGroup{},
	}
}

func (p *KeygenParty) Run(ctx context.Context) {
	params := bnb.NewParameters(
		bnb.S256(), bnb.NewPeerContext(p.sortedPartyIds),
		p.sortedPartyIds.FindByKey(p.self.Address.PartyKey()),
		len(p.sortedPartyIds),
		p.self.Threshold,
	)
	out := make(chan bnb.Message, tss.OutChannelSize)
	end := make(chan *keygen.LocalPartySaveData, tss.EndChannelSize)

	preParams, ok := p.self.PreParams.(keygen.LocalPreParams)
	if !ok {
		p.logger.WithError(errors.New("failed to convert types to LocalPreParams")).Error("failed to run keygen")
		close(end)
	}

	p.party = keygen.NewLocalParty(params, out, end, preParams)

	p.wg.Add(3)

	go func() {
		defer p.wg.Done()

		if err := p.party.Start(); err != nil {
			p.logger.WithError(err).Error("failed to run keygen")
			close(end)
		}
	}()

	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx, out, end)

	p.logger.Info("keygen started")
}

func (p *KeygenParty) WaitFor() *tss.LocalPartyData {
	p.wg.Wait()
	p.ended.Store(true)

	p.logger.Info("keygen finished")

	return tss.NewLocalPartyData(p.result)
}

func (p *KeygenParty) Receive(sender core.Address, data *p2p.TssData) {
	if p.ended.Load() {
		return
	}

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
				return
			}

			if _, exists := p.parties[msg.Sender]; !exists {
				p.logger.WithField("party", msg.Sender).Warn("got message from outside party")
				continue
			}

			_, err := p.party.UpdateFromBytes(msg.WireMsg, p.sortedPartyIds.FindByKey(msg.Sender.PartyKey()), msg.IsBroadcast)
			if err != nil {
				p.logger.WithError(err).Error("failed to update party state")
			}
		}
	}

}

func (p *KeygenParty) receiveUpdates(ctx context.Context, out <-chan bnb.Message, end <-chan *keygen.LocalPartySaveData) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping listening to updates")
			return

		case result, ok := <-end:
			if !ok {
				p.logger.Error("tss party result channel is closed")
			}

			close(p.msgs)
			p.result = result
			return

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
				Sender:    p.self.Address.String(),
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

			dst := core.AddrFromString(to[0].Moniker)
			if err = p.broadcaster.Send(&submitReq, dst); err != nil {
				p.logger.WithError(err).Error("failed to send message")
			}
		}
	}
}
