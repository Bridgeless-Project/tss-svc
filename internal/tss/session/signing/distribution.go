package signing

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/p2p/broadcast"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type SignaturesDistributor struct {
	wg *sync.WaitGroup

	sessionId   string
	distributor core.Address
	self        core.Address
	sigData     [][]byte
	sigPubKey   *ecdsa.PublicKey

	broadcaster broadcast.ReliableBroadcaster[tss.Signatures]

	initSigChan     chan broadcast.ReliableBroadcastMsg[tss.Signatures]
	initSigAccepted atomic.Bool

	signatures *tss.Signatures
	err        error

	logger *logan.Entry
}

func (s *SignaturesDistributor) Run(ctx context.Context) {
	s.wg.Add(1)

	if s.self == s.distributor {
		go s.distribute()
	} else {
		go s.receive(ctx)
	}
}

func (s *SignaturesDistributor) distribute() {
	defer s.wg.Done()

	if s.signatures == nil {
		s.err = errors.New("no signatures to distribute")
		return
	}

	if !s.broadcaster.Broadcast(s.signatures) {
		s.err = errors.New("signatures was not correctly broadcast")
		return
	}

	// additionally validating signatures
	if err := s.validateSignatures(s.signatures); err != nil {
		s.err = errors.Wrap(err, "failed to validate signatures")
		return
	}
}

func (s *SignaturesDistributor) receive(ctx context.Context) {
	defer s.wg.Done()

	var msg broadcast.ReliableBroadcastMsg[tss.Signatures]
	select {
	case <-ctx.Done():
		s.err = ctx.Err()
		return
	case msg = <-s.initSigChan:
		s.initSigAccepted.Store(true)
	}

	if !s.broadcaster.EnsureValid(msg) {
		s.err = errors.New("signatures message was not correctly distributed")
		return
	}

	signatures := msg.Msg.Value
	if signatures == nil {
		s.err = errors.New("no signatures received")
		return
	}
	if err := s.validateSignatures(signatures); err != nil {
		s.err = errors.Wrap(err, "failed to validate received signatures")
		return
	}
}

func (s *SignaturesDistributor) validateSignatures(signatures *tss.Signatures) error {
	if len(signatures.Data) != len(s.sigData) {
		return errors.New("received signatures count does not match expected")
	}

	for i, signature := range signatures.Data {
		if !tss.Verify(s.sigPubKey, s.sigData[i], signature) {
			return errors.New("got invalid signature")
		}
	}

	return nil
}

func (s *SignaturesDistributor) WaitFor() (*tss.Signatures, error) {
	s.wg.Wait()

	return s.signatures, s.err
}

func (s *SignaturesDistributor) Receive(request *p2p.SubmitRequest) error {
	if request == nil {
		return errors.New("nil request")
	}
	if request.SessionId != s.sessionId {
		return errors.New(fmt.Sprintf("session id mismatch: expected '%s', got '%s'", s.sessionId, request.SessionId))
	}
	if request.Type != p2p.RequestType_RT_SIGNATURE_DISTRIBUTION {
		return errors.New("invalid request type")
	}

	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}

	data := &p2p.ReliableBroadcastData{}
	if err = request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal reliable broadcast data")
	}

	roundMsg, err := broadcast.DecodeRoundMessage[tss.Signatures](data.GetRoundMsg())
	if err != nil {
		return errors.Wrap(err, "failed to decode round message")
	}
	msg := broadcast.ReliableBroadcastMsg[tss.Signatures]{
		Sender: sender,
		Msg:    roundMsg,
	}

	if roundMsg.Round != 0 {
		return s.broadcaster.Receive(msg)
	}

	if sender != s.distributor {
		return errors.New(fmt.Sprintf("sender %s is not distributor", sender))
	}
	if s.initSigAccepted.Load() {
		return errors.New("invalid message, signature already accepted")
	}
	s.initSigChan <- msg

	return nil
}
