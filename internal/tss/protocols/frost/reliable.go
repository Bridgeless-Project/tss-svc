package tss

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	roottss "github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
	"gitlab.com/distributed_lab/logan/v3"
)

type reliablePayload struct {
	Data []byte
}

func (p reliablePayload) HashString() string {
	return fmt.Sprintf("%x", sha256.Sum256(p.Data))
}

// reliableTransport runs one authenticated reliable-broadcast instance for
// each Taurus broadcast message. Taurus requires this property for messages
// marked Broadcast; ordinary P2P fanout is not sufficient under equivocation.
type reliableTransport struct {
	mu sync.Mutex

	sessionID   string
	requestType p2p.RequestType
	account     core.Account
	parties     []p2p.Party
	partySet    map[core.Address]struct{}
	threshold   int
	logger      *logan.Entry
	deliver     func(roottss.PartyMsg)

	instances map[string]*broadcast.ReliableBroadcaster[reliablePayload]
}

func newReliableTransport(
	sessionID string,
	requestType p2p.RequestType,
	account core.Account,
	parties []p2p.Party,
	threshold int,
	logger *logan.Entry,
	deliver func(roottss.PartyMsg),
) *reliableTransport {
	partySet := make(map[core.Address]struct{}, len(parties)+1)
	partySet[account.CosmosAddress()] = struct{}{}
	for _, remote := range parties {
		partySet[remote.CoreAddress] = struct{}{}
	}
	return &reliableTransport{
		sessionID:   sessionID,
		requestType: requestType,
		account:     account,
		parties:     parties,
		partySet:    partySet,
		threshold:   threshold,
		logger:      logger.WithField("component", "reliable-broadcast"),
		deliver:     deliver,
		instances:   make(map[string]*broadcast.ReliableBroadcaster[reliablePayload]),
	}
}

func (t *reliableTransport) broadcast(raw []byte) error {
	message, err := decodeFrostMessage(raw)
	if err != nil {
		return err
	}
	if !message.Broadcast || message.To != "" {
		return errors.New("attempted reliable broadcast of a direct FROST message")
	}
	if message.From != party.ID(t.account.CosmosAddress().String()) {
		return errors.New("attempted reliable broadcast with an invalid FROST sender")
	}
	if err := t.validateThreshold(); err != nil {
		return err
	}

	broadcastID := t.broadcastID(message)
	b := t.newInstance(broadcastID)
	if !t.addInstance(broadcastID, b) {
		return errors.Errorf("duplicate FROST reliable broadcast %q", broadcastID)
	}
	defer t.removeInstance(broadcastID, b)

	return b.Broadcast(&reliablePayload{Data: raw})
}

func (t *reliableTransport) receive(sender core.Address, data []byte) error {
	if err := t.validateThreshold(); err != nil {
		return err
	}
	if _, ok := t.partySet[sender]; !ok {
		return errors.Errorf("FROST reliable broadcast sender %s is not a participant", sender)
	}

	roundMessage, err := broadcast.DecodeRoundMessage[reliablePayload](broadcast.ReliableTSSPayload(data))
	if err != nil {
		return errors.Wrap(err, "failed to decode FROST reliable broadcast")
	}
	if roundMessage.Value == nil {
		return errors.New("FROST reliable broadcast contains no message")
	}

	message, err := decodeFrostMessage(roundMessage.Value.Data)
	if err != nil {
		return err
	}
	if !message.Broadcast || message.To != "" {
		return errors.New("FROST reliable broadcast contains a direct message")
	}
	wantID := t.broadcastID(message)
	if roundMessage.SessionId != wantID {
		return errors.Errorf("invalid FROST reliable broadcast id: expected %q, got %q", wantID, roundMessage.SessionId)
	}
	if len(roundMessage.Signatures) == 0 {
		return errors.New("FROST reliable broadcast contains no signatures")
	}
	origin := roundMessage.Signatures[0].Signer
	if _, ok := t.partySet[origin]; !ok {
		return errors.Errorf("FROST reliable broadcast origin %s is not a participant", origin)
	}
	if message.From != party.ID(origin.String()) {
		return errors.Errorf("FROST reliable broadcast origin mismatch: signed=%s embedded=%s", origin, message.From)
	}

	reliableMessage := broadcast.ReliableBroadcastMsg[reliablePayload]{
		Msg:    roundMessage,
		Sender: sender,
	}

	t.mu.Lock()
	b, exists := t.instances[wantID]
	if !exists {
		if len(t.instances) >= len(t.partySet)*8 {
			t.mu.Unlock()
			return errors.New("too many active FROST reliable broadcasts")
		}
		b = t.newInstance(wantID)
		t.instances[wantID] = b
	}
	t.mu.Unlock()

	if exists {
		return b.Receive(reliableMessage)
	}

	go func() {
		valid := b.EnsureValidFrom(origin, reliableMessage)
		t.removeInstance(wantID, b)
		if !valid {
			t.logger.WithField("broadcast_id", wantID).Warn("rejected FROST reliable broadcast")
			return
		}
		t.deliver(roottss.PartyMsg{
			Sender:      origin,
			WireMsg:     roundMessage.Value.Data,
			IsBroadcast: true,
		})
	}()

	return nil
}

func (t *reliableTransport) newInstance(broadcastID string) *broadcast.ReliableBroadcaster[reliablePayload] {
	return broadcast.NewReliableForTSSRoute[reliablePayload](
		broadcastID,
		t.sessionID,
		t.parties,
		t.account,
		t.threshold,
		t.requestType,
		t.logger.WithField("broadcast_id", broadcastID),
	)
}

func (t *reliableTransport) broadcastID(message *protocol.Message) string {
	return fmt.Sprintf(
		"FROST_RBC/%s/%s/%d/%s",
		t.sessionID,
		message.Protocol,
		message.RoundNumber,
		message.From,
	)
}

func (t *reliableTransport) validateThreshold() error {
	partyCount := len(t.parties) + 1
	if t.threshold < 0 || t.threshold >= partyCount {
		return errors.Errorf("invalid FROST threshold %d for %d parties", t.threshold, partyCount)
	}
	return nil
}

func (t *reliableTransport) addInstance(id string, instance *broadcast.ReliableBroadcaster[reliablePayload]) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.instances[id]; exists {
		return false
	}
	t.instances[id] = instance
	return true
}

func (t *reliableTransport) removeInstance(id string, instance *broadcast.ReliableBroadcaster[reliablePayload]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.instances[id] == instance {
		delete(t.instances, id)
	}
}

func decodeFrostMessage(raw []byte) (*protocol.Message, error) {
	message := new(protocol.Message)
	if err := message.UnmarshalBinary(raw); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal FROST message")
	}
	if message.Protocol == "" || message.From == "" || message.RoundNumber <= 0 {
		return nil, errors.New("invalid empty FROST message envelope")
	}
	return message, nil
}
