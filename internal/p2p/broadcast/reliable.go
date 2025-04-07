package broadcast

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// TODO: enough or too much?
	roundTimeout = 500 * time.Millisecond

	defaultChanCapacity = 200
)

type Hashable interface {
	HashString() string
}

type Signature struct {
	Signer core.Address
	Value  []byte
}

type RoundMessage[T Hashable] struct {
	Value     *T
	SessionId string

	Round      int
	Signatures []Signature
}

func (m RoundMessage[T]) Encode() []byte {
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	_ = encoder.Encode(m)

	return buff.Bytes()
}

func DecodeRoundMessage[T Hashable](data []byte) (RoundMessage[T], error) {
	var msg RoundMessage[T]

	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&msg); err != nil {
		return RoundMessage[T]{}, errors.Wrap(err, "failed to decode round message")
	}

	return msg, nil
}

func (m RoundMessage[T]) SignatureValid(signature Signature) bool {
	pubKey, err := crypto.SigToPub(m.SignHash(), signature.Value)
	if err != nil {
		return false
	}
	recovered := crypto.PubkeyToAddress(*pubKey)

	return bytes.Equal(recovered.Bytes(), signature.Signer.Bytes())
}

func (m RoundMessage[T]) SignHash() []byte {
	var buf bytes.Buffer

	encoder := gob.NewEncoder(&buf)
	_ = encoder.Encode(m.SessionId)
	// gob cannot encode nil values, but they are valid
	if m.Value != (*T)(nil) {
		_ = encoder.Encode(m.Value)
	}

	for _, sig := range m.Signatures {
		_ = encoder.Encode(sig.Signer)
		_ = encoder.Encode(sig.Value)
	}

	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

type ReliableBroadcastMsg[T Hashable] struct {
	Msg    RoundMessage[T]
	Sender core.Address
}

// ReliableBroadcaster is a reliable broadcast protocol implementation based on the Dolev-Strong protocol.
// It ensures that a same message is delivered to all parties in the group.
// It is designed to work in a synchronous network with n > t,
// where n is the number of parties and t is the maximum number of malicious parties.
//
// Instead of running the relay rounds one by one, it runs one big round and processes all incoming messages,
// ensuring each early or late but valid message is processed.
type ReliableBroadcaster[T Hashable] struct {
	sessionId   string
	parties     []p2p.Party
	self        core.Account
	logger      *logan.Entry
	requestType p2p.RequestType

	relayRounds int
	broadcaster *Broadcaster
	partiesMap  map[core.Address]bool

	originMsgSender core.Address

	// sender -> round -> received
	receivedMsgs         map[core.Address]map[int]bool
	values               map[string]*T
	msgs                 chan ReliableBroadcastMsg[T]
	finalSigChainReached bool
}

func NewReliable[T Hashable](
	sessionId string,
	parties []p2p.Party,
	self core.Account,
	threshold int,
	requestType p2p.RequestType,
	logger *logan.Entry,
) *ReliableBroadcaster[T] {
	// relay rounds are calculated as the t + 1,
	// where t is the maximum number of possible malicious parties,
	// which is equal to the total number of parties minus the threshold
	relayRounds := (len(parties) + 1) - threshold + 1

	receivedMsgsMap := make(map[core.Address]map[int]bool, len(parties))
	partiesMap := make(map[core.Address]bool, len(parties)+1)
	for _, party := range parties {
		receivedMsgsMap[party.CoreAddress] = make(map[int]bool, relayRounds)
		partiesMap[party.CoreAddress] = true
	}
	partiesMap[self.CosmosAddress()] = true

	return &ReliableBroadcaster[T]{
		sessionId:   sessionId,
		parties:     parties,
		self:        self,
		logger:      logger,
		requestType: requestType,

		relayRounds: relayRounds,
		broadcaster: NewBroadcaster(parties, logger),
		partiesMap:  partiesMap,

		receivedMsgs: receivedMsgsMap,
		values:       make(map[string]*T, 1),
		msgs:         make(chan ReliableBroadcastMsg[T], defaultChanCapacity),
	}
}

func (b *ReliableBroadcaster[T]) Broadcast(msg *T) bool {
	b.addToValuesSet(msg)
	b.originMsgSender = b.self.CosmosAddress()

	roundMsg := RoundMessage[T]{
		SessionId: b.sessionId,
		Value:     msg,
		Round:     0,
	}
	signHash := roundMsg.SignHash()
	sig, err := b.self.PrivateKey().Sign(signHash)
	if err != nil {
		b.logger.Warn(fmt.Sprintf("failed to sign initial broadcasting message: %s", err))
		return false
	}
	roundMsg.Signatures = []Signature{{Signer: b.self.CosmosAddress(), Value: sig}}

	b.broadcastMsg(roundMsg)

	// validating the incoming round messages from parties
	b.startRounds()

	// checking if the message was delivered successfully
	return b.decideValid()
}

func (b *ReliableBroadcaster[T]) EnsureValid(msg ReliableBroadcastMsg[T]) bool {
	b.originMsgSender = msg.Sender
	b.msgs <- msg

	b.startRounds()

	return b.decideValid()
}

func (b *ReliableBroadcaster[T]) Receive(msg ReliableBroadcastMsg[T]) error {
	if !b.partiesMap[msg.Sender] {
		return errors.New("party is not in the group")
	}

	b.msgs <- msg

	return nil
}

func (b *ReliableBroadcaster[T]) startRounds() {
	// excluding the first round, which is the sender's message
	roundsDuration := time.Duration(b.relayRounds) * roundTimeout
	ctx, cancel := context.WithTimeout(context.Background(), roundsDuration)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-b.msgs:
			if msg.Msg.SessionId != b.sessionId {
				b.logger.Warn(fmt.Sprintf("malicious party %q sending message with different session id", msg.Sender))
				continue
			}
			if msg.Msg.Round > b.relayRounds {
				b.logger.Warn(fmt.Sprintf("malicious party %q sending message with round greater than relay rounds count", msg.Sender))
				continue
			}
			if b.receivedMsgs[msg.Sender][msg.Msg.Round] {
				b.logger.Warn(fmt.Sprintf("malicious party %q sending duplicate round message", msg.Sender))
				continue
			}
			b.receivedMsgs[msg.Sender][msg.Msg.Round] = true

			b.processMsg(msg)
		}
	}
}

func (b *ReliableBroadcaster[T]) processMsg(msg ReliableBroadcastMsg[T]) {
	signaturesValid, selfSigned := b.validateSignatures(msg)
	if !signaturesValid {
		return
	}

	b.addToValuesSet(msg.Msg.Value)
	if msg.Msg.Round == b.relayRounds {
		b.finalSigChainReached = true
		return
	}

	// no need to sign and/or broadcast messages already signed by self
	if selfSigned {
		return
	}

	signHash := msg.Msg.SignHash()
	sig, err := b.self.PrivateKey().Sign(signHash)
	if err != nil {
		b.logger.Warn(fmt.Sprintf("failed to sign message: %s", err))
		return
	}

	msg.Msg.Signatures = append(msg.Msg.Signatures, Signature{
		Signer: b.self.CosmosAddress(),
		Value:  sig,
	})
	msg.Msg.Round += 1

	b.broadcastMsg(msg.Msg)
}

func (b *ReliableBroadcaster[T]) decideValid() bool {
	distinctValuesCount := len(b.values)
	if distinctValuesCount == 0 || distinctValuesCount > 1 {
		b.logger.Warn("no valid values found or too many distinct values")
		return false
	}

	if !b.finalSigChainReached {
		b.logger.Warn("longest signature chain not reached, too much malicious parties")
		return false
	}

	return true
}

// empty value can also be valid
func (b *ReliableBroadcaster[T]) addToValuesSet(value *T) {
	if value == (*T)(nil) {
		if _, ok := b.values[""]; !ok {
			b.values[""] = value
		}

		return
	}

	if _, ok := b.values[(*value).HashString()]; !ok {
		b.values[(*value).HashString()] = value
	}
}

func (b *ReliableBroadcaster[T]) broadcastMsg(msg RoundMessage[T]) {
	rawReq, _ := anypb.New(&p2p.ReliableBroadcastData{RoundMsg: msg.Encode()})
	b.broadcaster.Broadcast(&p2p.SubmitRequest{
		Sender:    b.self.CosmosAddress().String(),
		SessionId: b.sessionId,
		Type:      b.requestType,
		Data:      rawReq,
	})
}

func (b *ReliableBroadcaster[T]) validateSignatures(msg ReliableBroadcastMsg[T]) (valid, selfSigned bool) {
	roundMsg := msg.Msg
	if len(roundMsg.Signatures) != roundMsg.Round+1 {
		b.logger.Warn(fmt.Sprintf("malicious party %q sending incomplete signature chain", msg.Sender))
		return
	}

	// the first signature in the chain must be from the msg broadcaster
	// the last signature in the chain must be from the original sender
	// the rest must be from the distinct parties in the group
	tempMsg := roundMsg
	senderChecked := make(map[core.Address]bool, len(roundMsg.Signatures))
	for idx, signature := range slices.Backward(roundMsg.Signatures) {
		if !b.partiesMap[signature.Signer] {
			b.logger.Warn(fmt.Sprintf("malicious party %q sending message with invalid signer", msg.Sender))
			return false, selfSigned
		}
		if idx == 0 && signature.Signer != b.originMsgSender {
			b.logger.Warn(fmt.Sprintf("malicious party %q sending message with invalid first signer", msg.Sender))
			return false, selfSigned
		}
		if idx == len(roundMsg.Signatures)-1 && signature.Signer != msg.Sender {
			b.logger.Warn(fmt.Sprintf("malicious party %q sending message with invalid last signer", msg.Sender))
			return false, selfSigned
		}
		if senderChecked[signature.Signer] {
			b.logger.Warn(fmt.Sprintf("malicious party %q sending message with duplicate signer", msg.Sender))
			return false, selfSigned
		}

		// popping the last signature
		tempMsg.Signatures = tempMsg.Signatures[:len(tempMsg.Signatures)-1]
		if !tempMsg.SignatureValid(signature) {
			b.logger.Warn(fmt.Sprintf("malicious party %q sending message with invalid signature", msg.Sender))
			return false, selfSigned
		}

		senderChecked[signature.Signer] = true
		if signature.Signer == b.self.CosmosAddress() {
			selfSigned = true
		}
	}

	return true, selfSigned
}
