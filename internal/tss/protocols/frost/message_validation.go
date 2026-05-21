package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	rootTss "github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
	"github.com/taurusgroup/multi-party-sig/pkg/protocol"
)

func validateMessageEnvelope(sender core.Address, selfID party.ID, msg rootTss.PartyMsg, message *protocol.Message) error {
	if message == nil {
		return errors.New("nil FROST message")
	}

	if message.From != party.ID(sender.String()) {
		return errors.Errorf("FROST message sender mismatch: envelope=%s embedded=%s", sender, message.From)
	}

	if message.Broadcast != msg.IsBroadcast {
		return errors.Errorf("FROST message broadcast mismatch: envelope=%t embedded=%t", msg.IsBroadcast, message.Broadcast)
	}

	if msg.IsBroadcast {
		if message.To != "" {
			return errors.Errorf("FROST broadcast message has direct recipient: %s", message.To)
		}
		return nil
	}

	if message.To != selfID {
		return errors.Errorf("FROST direct message is addressed to %s, expected %s", message.To, selfID)
	}

	return nil
}
