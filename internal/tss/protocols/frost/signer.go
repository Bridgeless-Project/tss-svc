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
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
	"github.com/taurusgroup/multi-party-sig/protocols/frost"
	"github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

// TODO remove unused log after tests
type SignParty struct {
	wg    *sync.WaitGroup
	ended atomic.Bool

	broadcaster *broadcast.Broadcaster
	reliable    *reliableTransport
	parties     map[core.Address]struct{}
	signers     []party.ID

	self tss.LocalSignParty

	group curve.Curve

	sessionId string
	handler   *protocol.MultiHandler

	msgs   chan tss.PartyMsg
	done   chan struct{}
	once   sync.Once
	data   []byte
	result tss.SignatureData
	err    error
	logger *logan.Entry
}

func NewSignParty(self tss.LocalSignParty, sessionId string, logger *logan.Entry) *SignParty {
	return &SignParty{
		wg:        new(sync.WaitGroup),
		self:      self,
		msgs:      make(chan tss.PartyMsg, tss.MsgsCapacity),
		done:      make(chan struct{}),
		sessionId: sessionId,
		logger:    logger.WithField("protocol", "frost"),
		group:     self.Share.Group(),
		result:    new(FrostSignature),
	}
}

func (p *SignParty) WithParties(parties []p2p.Party) tss.SignParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	signers := make([]party.ID, 0, len(parties)+1)
	signers = append(signers, party.ID((p.self.Account.CosmosAddress().String())))

	for _, p2pParty := range parties {
		partyMap[p2pParty.CoreAddress] = struct{}{}
		signers = append(signers, party.ID(p2pParty.CoreAddress.String()))
	}

	p.parties = partyMap
	p.signers = party.NewIDSlice(signers)
	p.broadcaster = broadcast.NewBroadcaster(parties, p.logger.WithField("component", "broadcaster"))
	p.reliable = newReliableTransport(
		p.sessionId,
		p2p.RequestType_RT_SIGN,
		p.self.Account,
		parties,
		p.self.Threshold,
		p.logger,
		p.enqueue,
	)

	return p
}

func (p *SignParty) WithSigningData(data []byte) tss.SignParty {
	p.data = data
	return p
}

func (p *SignParty) Run(ctx context.Context) {
	frostShare, ok := p.self.Share.(*FrostShare)
	if !ok {
		p.err = errors.New("invalid FROST share implementation")
		p.finish()
		return
	}
	if err := frostShare.Validate(
		party.ID(p.self.Account.CosmosAddress().String()),
		p.self.Threshold,
		p.signers,
	); err != nil {
		p.err = fmt.Errorf("invalid FROST signing share: %w", err)
		p.finish()
		p.logger.WithError(p.err).Error("failed to validate FROST signing share")
		return
	}

	config, err := toTaprootConfig(frostShare.MustFrostShare())
	if err != nil {
		p.err = err
		p.finish()
		p.logger.WithError(err).Error("failed to prepare frost signing config")
		return
	}

	h, err := protocol.NewMultiHandler(frost.SignTaproot(config, p.signers, p.data), []byte(p.sessionId))
	if err != nil {
		p.err = err
		p.finish()
		p.logger.WithError(err).Error("failed to create frost signing handler")
		return
	}
	p.handler = h

	p.wg.Add(2)
	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx)

	p.logger.Info("frost signing started")
}

func (p *SignParty) WaitFor() tss.SignatureData {
	p.wg.Wait()
	p.finish()

	p.logger.Info("frost signing finished")

	if p.err != nil {
		p.logger.Debug("frost signing failed")
		return nil
	}

	return p.result
}

func (p *SignParty) Receive(sender core.Address, data *p2p.TssData) {
	if p == nil || data == nil || p.ended.Load() {
		return
	}
	if broadcast.IsReliableTSSData(data.Data) {
		if p.reliable == nil {
			p.logger.Warn("received FROST reliable broadcast before parties were configured")
			return
		}
		if err := p.reliable.receive(sender, data.Data); err != nil {
			p.logger.WithError(err).WithField("party", sender).Warn("rejected FROST reliable broadcast message")
		}
		return
	}
	if data.IsBroadcast {
		p.logger.WithField("party", sender).Warn("rejected FROST broadcast outside reliable transport")
		return
	}

	p.enqueue(tss.PartyMsg{
		Sender:      sender,
		WireMsg:     data.Data,
		IsBroadcast: data.IsBroadcast,
	})
}

func (p *SignParty) enqueue(msg tss.PartyMsg) {
	select {
	case <-p.done:
		return
	case p.msgs <- msg:
		return
	default:
		p.logger.WithField("party", msg.Sender).Warn("FROST signing inbox is full; rejecting message")
	}
}

func (p *SignParty) receiveMsgs(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping receiving frost messages")
			return
		case <-p.done:
			return
		case msg := <-p.msgs:

			if _, exists := p.parties[msg.Sender]; !exists {
				p.logger.WithField("party", msg.Sender).Warn("got message from outside party")
				continue
			}

			message := new(protocol.Message)
			if err := message.UnmarshalBinary(msg.WireMsg); err != nil {
				p.logger.WithError(err).WithField("party", msg.Sender).Warn("failed to unmarshal frost message")
				continue
			}
			if err := validateMessageEnvelope(msg.Sender, party.ID(p.self.Account.CosmosAddress().String()), msg, message); err != nil {
				p.logger.WithError(err).WithField("party", msg.Sender).Warn("rejected invalid frost message envelope")
				continue
			}

			p.handler.Accept(message)
		}
	}
}

func (p *SignParty) receiveUpdates(ctx context.Context) {
	defer func() { p.wg.Done(); p.finish() }()

	for {
		select {
		case <-ctx.Done():
			p.err = ctx.Err()
			p.logger.Warn("context is done; stopping listening to frost updates")
			return
		case msg, ok := <-p.handler.Listen():
			if !ok {
				result, err := p.handler.Result()
				if err != nil {
					p.err = err
					p.logger.WithError(err).Error("failed to get frost signing result")
					return
				}

				signature, ok := result.(taproot.Signature)
				if !ok {
					p.err = errors.New("unexpected frost signing result type")
					p.logger.WithField("type", result).Error("failed to get frost signing result")
					return
				}
				// convert signature to bytes before passing to SetSignature
				err = p.result.SetSignature([]byte(signature))
				if err != nil {
					p.err = err
				}

				return
			}

			raw, err := msg.MarshalBinary()
			if err != nil {
				p.logger.WithError(err).Error("failed to marshal frost message")
				continue
			}

			if msg.Broadcast {
				if p.reliable == nil {
					p.err = errors.New("FROST reliable broadcast is not configured")
					return
				}
				if err = p.reliable.broadcast(raw); err != nil {
					p.err = fmt.Errorf("failed to reliably broadcast FROST signing message: %w", err)
					p.logger.WithError(p.err).Error("failed to send FROST signing message")
					return
				}
				continue
			}

			tssData := &p2p.TssData{
				Data:        raw,
				IsBroadcast: false,
			}

			tssReq, err := anypb.New(tssData)
			if err != nil {
				p.err = fmt.Errorf("failed to encode FROST signing message: %w", err)
				return
			}
			submitReq := p2p.SubmitRequest{
				Sender:    p.self.Account.CosmosAddress().String(),
				SessionId: p.sessionId,
				Type:      p2p.RequestType_RT_SIGN,
				Data:      tssReq,
			}

			if msg.To == "" {
				p.broadcaster.Broadcast(&submitReq)
				continue
			}

			dst := core.AddrFromString(string(msg.To))
			if err := p.broadcaster.Send(&submitReq, dst); err != nil {
				p.logger.WithError(err).Error("failed to send frost message")
			}
		}
	}
}

func (p *SignParty) finish() {
	p.once.Do(func() {
		p.ended.Store(true)
		close(p.done)
	})
}

func toTaprootConfig(config *keygen.Config) (*keygen.TaprootConfig, error) {
	if config == nil {
		return nil, errors.New("missing frost share")
	}

	privateShare, ok := config.PrivateShare.(*curve.Secp256k1Scalar)
	if !ok {
		return nil, errors.New("frost private share is not secp256k1")
	}

	publicKey, ok := config.PublicKey.(*curve.Secp256k1Point)
	if !ok {
		return nil, errors.New("frost public key is not secp256k1")
	}

	privateShare = curve.Secp256k1{}.NewScalar().Set(privateShare).(*curve.Secp256k1Scalar)
	verificationShares := make(map[party.ID]*curve.Secp256k1Point, len(config.VerificationShares.Points))
	for id, point := range config.VerificationShares.Points {
		secpPoint, ok := point.(*curve.Secp256k1Point)
		if !ok {
			return nil, errors.New("frost verification share is not secp256k1")
		}
		verificationShares[id] = secpPoint
	}

	if !publicKey.HasEvenY() {
		privateShare.Negate()
		for id, point := range verificationShares {
			verificationShares[id] = point.Negate().(*curve.Secp256k1Point)
		}
	}

	return &keygen.TaprootConfig{
		ID:                 config.ID,
		Threshold:          config.Threshold,
		PrivateShare:       privateShare,
		PublicKey:          publicKey.XBytes(),
		ChainKey:           config.ChainKey,
		VerificationShares: verificationShares,
	}, nil
}
