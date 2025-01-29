package tss

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

type LocalKeygenParty struct {
	PreParams keygen.LocalPreParams
	Address   core.Address
}

type KeygenParty struct {
	wg    *sync.WaitGroup
	ended atomic.Bool

	broadcaster    *p2p.Broadcaster
	party          tss.Party
	sortedPartyIds tss.SortedPartyIDs
	parties        map[core.Address]struct{}
	self           LocalKeygenParty

	msgs      chan partyMsg
	result    *keygen.LocalPartySaveData
	sessionId string

	logger *logan.Entry
}

func NewKeygenParty(self LocalKeygenParty, parties []p2p.Party, sessionId string, logger *logan.Entry) *KeygenParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyIds := make([]*tss.PartyID, len(parties)+1)
	partyIds[0] = self.Address.PartyIdentifier()

	for i, party := range parties {
		partyMap[party.CoreAddress] = struct{}{}
		partyIds[i+1] = party.Identifier()
	}

	return &KeygenParty{
		broadcaster:    p2p.NewBroadcaster(parties),
		sortedPartyIds: tss.SortPartyIDs(partyIds),
		parties:        partyMap,
		self:           self,
		msgs:           make(chan partyMsg, MsgsCapacity),
		logger:         logger,
		sessionId:      sessionId,
		wg:             &sync.WaitGroup{},
	}
}

func (p *KeygenParty) Run(ctx context.Context) {
	params := tss.NewParameters(
		tss.S256(), tss.NewPeerContext(p.sortedPartyIds),
		p.sortedPartyIds.FindByKey(p.self.Address.PartyKey()),
		len(p.sortedPartyIds),
		getKeygenThreshold(len(p.sortedPartyIds)),
	)
	out := make(chan tss.Message, OutChannelSize)
	end := make(chan *keygen.LocalPartySaveData, EndChannelSize)

	p.party = keygen.NewLocalParty(params, out, end, p.self.PreParams)

	p.wg.Add(3)

	go func() {
		defer p.wg.Done()

		if err := p.party.Start(); err != nil {
			p.logger.WithError(err).Error("failed to run keygen party")
			close(end)
		}
	}()
	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx, out, end)
}

func (p *KeygenParty) WaitFor() *keygen.LocalPartySaveData {
	p.wg.Wait()
	p.ended.Store(true)

	return p.result
}

func (p *KeygenParty) Receive(sender core.Address, data *p2p.TssData) {
	if p.ended.Load() {
		return
	}

	p.msgs <- partyMsg{
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
				p.logger.Debug("msg channel is closed")
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

func (p *KeygenParty) receiveUpdates(ctx context.Context, out <-chan tss.Message, end <-chan *keygen.LocalPartySaveData) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping listening to updates")
			return
		case result, ok := <-end:
			close(p.msgs)
			p.result = result

			if !ok {
				p.logger.Error("tss party result channel is closed")
			}

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

			dst := core.AddrFromPartyId(to[0])
			if err = p.broadcaster.Send(&submitReq, dst); err != nil {
				p.logger.WithError(err).Error("failed to send message")
			}
		}
	}
}

func getKeygenThreshold(partiesNum int) int {
	return int(float32(partiesNum) * 2 / 3)
}
