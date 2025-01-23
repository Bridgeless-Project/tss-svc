package tss

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

type LocalSignParty struct {
	Address   core.Address
	Data      *keygen.LocalPartySaveData
	Threshold int
}

type SignParty struct {
	wg *sync.WaitGroup

	parties        map[core.Address]struct{}
	sortedPartyIds tss.SortedPartyIDs

	self LocalSignParty

	logger      *logan.Entry
	party       tss.Party
	msgs        chan partyMsg
	broadcaster *p2p.Broadcaster

	data []byte

	ended     atomic.Bool
	result    *common.SignatureData
	sessionId string
}

func NewSignParty(self LocalSignParty, parties []p2p.Party, data []byte, sessionId string, logger *logan.Entry) *SignParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	partyIds := make([]*tss.PartyID, len(parties)+1)
	partyIds[0] = self.Address.PartyIdentifier()

	for i, party := range parties {
		if party.CoreAddress == self.Address {
			continue
		}

		partyMap[party.CoreAddress] = struct{}{}
		partyIds[i+1] = party.Identifier()
	}
	return &SignParty{
		wg:             &sync.WaitGroup{},
		self:           self,
		sortedPartyIds: tss.SortPartyIDs(partyIds),
		parties:        partyMap,
		data:           data,
		msgs:           make(chan partyMsg, MsgsCapacity),
		sessionId:      sessionId,
		logger:         logger,
		broadcaster:    p2p.NewBroadcaster(parties),
	}
}

func (p *SignParty) Run(ctx context.Context) {
	p.logger.Infof("Running TSS signing on set: %v", p.parties)
	params := tss.NewParameters(
		tss.S256(), tss.NewPeerContext(p.sortedPartyIds),
		p.sortedPartyIds.FindByKey(p.self.Address.PartyKey()),
		len(p.sortedPartyIds),
		p.self.Threshold,
	)
	out := make(chan tss.Message, OutChannelSize)
	end := make(chan *common.SignatureData, EndChannelSize)

	p.party = signing.NewLocalParty(new(big.Int).SetBytes(p.data), params, *p.self.Data, out, end)

	p.wg.Add(3)

	go func() {
		defer p.wg.Done()
		if err := p.party.Start(); err != nil {
			p.logger.WithError(err).Error("failed to run signer party")
			close(end)
		}
	}()
	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx, out, end)
}

func (p *SignParty) WaitFor() *common.SignatureData {
	p.wg.Wait()
	p.ended.Store(true)
	return p.result
}

// Receive adds msg to msgs chan
func (p *SignParty) Receive(sender core.Address, data *p2p.TssData) {
	if p.ended.Load() {
		return
	}

	p.msgs <- partyMsg{
		Sender:      sender,
		WireMsg:     data.Data,
		IsBroadcast: data.IsBroadcast,
	}
}

// receiveMsgs receives message from msg chan and updates party`s internal state
func (p *SignParty) receiveMsgs(ctx context.Context) {
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

func (p *SignParty) receiveUpdates(ctx context.Context, out <-chan tss.Message, end <-chan *common.SignatureData) {
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
				Type:      p2p.RequestType_RT_SIGN,
				Data:      tssReq,
			}

			destination := routing.To
			if destination == nil || len(destination) > 1 {
				if err = p.broadcaster.Broadcast(&submitReq); err != nil {
					p.logger.WithError(err).Error("failed to broadcast message")
				}
				continue
			}

			dst := core.AddrFromPartyId(destination[0])
			if len(dst.String()) == 0 {
				p.logger.WithError(err).Error("failed to get destination address")
				continue
			}

			if err = p.broadcaster.Send(&submitReq, dst); err != nil {
				p.logger.WithError(err).Error("failed to send message")
			}
		}
	}
}
