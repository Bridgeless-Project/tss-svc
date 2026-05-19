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
	"github.com/bnb-chain/tss-lib/v2/common"
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
	parties     map[core.Address]struct{}
	signers     []party.ID

	self tss.LocalSignParty

	group curve.Curve

	sessionId string
	handler   *protocol.MultiHandler

	msgs   chan tss.PartyMsg
	data   []byte
	result *common.SignatureData
	err    error
	logger *logan.Entry
}

func NewSignParty(self tss.LocalSignParty, sessionId string, group curve.Curve, logger *logan.Entry) *SignParty {
	return &SignParty{
		wg:        &sync.WaitGroup{},
		self:      self,
		msgs:      make(chan tss.PartyMsg, tss.MsgsCapacity),
		sessionId: sessionId,
		logger:    logger.WithField("protocol", "frost"),
		group:     group,
	}
}

func (p *SignParty) WithParties(parties []p2p.Party) tss.SignParty {
	partyMap := make(map[core.Address]struct{}, len(parties))
	signers := make([]party.ID, 0, len(parties)+1)
	signers = append(signers, party.ID(p.self.Account.CosmosAddress().String()))

	for _, p2pParty := range parties {
		partyMap[p2pParty.CoreAddress] = struct{}{}
		signers = append(signers, party.ID(p2pParty.CoreAddress.String()))
	}

	p.parties = partyMap
	p.signers = party.NewIDSlice(signers)
	p.broadcaster = broadcast.NewBroadcaster(parties, p.logger.WithField("component", "broadcaster"))

	return p
}

func (p *SignParty) WithSigningData(data []byte) tss.SignParty {
	p.data = data
	return p
}

func (p *SignParty) Run(ctx context.Context) {
	//  TODO add custom curve here
	config, err := toTaprootConfig(p.self.FrostShare)
	if err != nil {
		p.err = err
		p.ended.Store(true)
		p.logger.WithError(err).Error("failed to prepare frost signing config")
		return
	}

	h, err := protocol.NewMultiHandler(frost.SignTaproot(config, p.signers, p.data), []byte(p.sessionId))
	if err != nil {
		p.err = err
		p.ended.Store(true)
		p.logger.WithError(err).Error("failed to create frost signing handler")
		return
	}
	p.handler = h

	p.wg.Add(2)
	go p.receiveMsgs(ctx)
	go p.receiveUpdates(ctx)

	p.logger.Info("frost signing started")
}

func (p *SignParty) WaitFor() *common.SignatureData {
	p.wg.Wait()
	p.ended.Store(true)

	p.logger.Info("frost signing finished")

	if p.err != nil {
		p.logger.Debug("frost signing failed")
		return nil
	}

	return p.result
}

func (p *SignParty) Receive(sender core.Address, data *p2p.TssData) {
	if p.ended.Load() {
		return
	}

	p.msgs <- tss.PartyMsg{
		Sender:      sender,
		WireMsg:     data.Data,
		IsBroadcast: data.IsBroadcast,
	}
}

func (p *SignParty) receiveMsgs(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping receiving frost messages")
			return
		case msg, ok := <-p.msgs:
			if !ok {
				return
			}

			if _, exists := p.parties[msg.Sender]; !exists {
				p.logger.WithField("party", msg.Sender).Warn("got message from outside party")
				continue
			}

			message := new(protocol.Message)
			if err := message.UnmarshalBinary(msg.WireMsg); err != nil {
				p.logger.WithError(err).WithField("party", msg.Sender).Warn("failed to unmarshal frost message")
				continue
			}

			p.handler.Accept(message)
		}
	}
}

func (p *SignParty) receiveUpdates(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("context is done; stopping listening to frost updates")
			return
		case msg, ok := <-p.handler.Listen():
			fmt.Println("got frost update", msg)
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

				p.result = frostSignatureData(signature, p.data)
				fmt.Println("got frost signing result", hex.EncodeToString(p.result.Signature))
				p.ended.Store(true)
				close(p.msgs)
				return
			}

			raw, err := msg.MarshalBinary()
			if err != nil {
				p.logger.WithError(err).Error("failed to marshal frost message")
				continue
			}

			tssData := &p2p.TssData{
				Data:        raw,
				IsBroadcast: msg.Broadcast,
			}

			tssReq, _ := anypb.New(tssData)
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

func frostSignatureData(signature taproot.Signature, msg []byte) *common.SignatureData {
	data := make([]byte, len(signature))
	copy(data, signature)

	result := &common.SignatureData{
		Signature: data,
		M:         append([]byte(nil), msg...),
	}
	if len(data) == taproot.SignatureLen {
		result.R = append([]byte(nil), data[:32]...)
		result.S = append([]byte(nil), data[32:]...)
	}

	return result
}
