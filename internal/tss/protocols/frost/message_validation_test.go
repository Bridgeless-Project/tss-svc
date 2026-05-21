package tss

import (
	"strings"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	rootTss "github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

func TestValidateMessageEnvelopeAcceptsValidDirectMessage(t *testing.T) {
	sender := core.Address("bridge1sender")
	selfID := party.ID("bridge1self")
	msg := rootTss.PartyMsg{Sender: sender, IsBroadcast: false}
	message := &protocol.Message{
		From:      party.ID(sender.String()),
		To:        selfID,
		Broadcast: false,
	}

	if err := validateMessageEnvelope(sender, selfID, msg, message); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageEnvelopeAcceptsValidBroadcastMessage(t *testing.T) {
	sender := core.Address("bridge1sender")
	selfID := party.ID("bridge1self")
	msg := rootTss.PartyMsg{Sender: sender, IsBroadcast: true}
	message := &protocol.Message{
		From:      party.ID(sender.String()),
		Broadcast: true,
	}

	if err := validateMessageEnvelope(sender, selfID, msg, message); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageEnvelopeRejectsMismatchedSender(t *testing.T) {
	sender := core.Address("bridge1sender")
	err := validateMessageEnvelope(
		sender,
		party.ID("bridge1self"),
		rootTss.PartyMsg{Sender: sender},
		&protocol.Message{From: party.ID("bridge1other"), To: party.ID("bridge1self")},
	)

	if err == nil || !strings.Contains(err.Error(), "sender mismatch") {
		t.Fatalf("expected sender mismatch error, got %v", err)
	}
}

func TestValidateMessageEnvelopeRejectsBroadcastMismatch(t *testing.T) {
	sender := core.Address("bridge1sender")
	err := validateMessageEnvelope(
		sender,
		party.ID("bridge1self"),
		rootTss.PartyMsg{Sender: sender, IsBroadcast: true},
		&protocol.Message{From: party.ID(sender.String()), Broadcast: false},
	)

	if err == nil || !strings.Contains(err.Error(), "broadcast mismatch") {
		t.Fatalf("expected broadcast mismatch error, got %v", err)
	}
}

func TestValidateMessageEnvelopeRejectsBroadcastWithDirectRecipient(t *testing.T) {
	sender := core.Address("bridge1sender")
	err := validateMessageEnvelope(
		sender,
		party.ID("bridge1self"),
		rootTss.PartyMsg{Sender: sender, IsBroadcast: true},
		&protocol.Message{From: party.ID(sender.String()), To: party.ID("bridge1self"), Broadcast: true},
	)

	if err == nil || !strings.Contains(err.Error(), "direct recipient") {
		t.Fatalf("expected direct recipient error, got %v", err)
	}
}

func TestValidateMessageEnvelopeRejectsDirectMessageForAnotherParty(t *testing.T) {
	sender := core.Address("bridge1sender")
	err := validateMessageEnvelope(
		sender,
		party.ID("bridge1self"),
		rootTss.PartyMsg{Sender: sender, IsBroadcast: false},
		&protocol.Message{From: party.ID(sender.String()), To: party.ID("bridge1other"), Broadcast: false},
	)

	if err == nil || !strings.Contains(err.Error(), "addressed to") {
		t.Fatalf("expected addressed-to error, got %v", err)
	}
}
