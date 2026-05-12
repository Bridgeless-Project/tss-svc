package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	bnb "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ECDSA"
	frost "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/FROST"
	"gitlab.com/distributed_lab/logan/v3"
)

func SelectKeyGenByProtocol(protocol int, self tss.LocalKeygenParty, parties []p2p.Party, threshold int, sessionId string, logger *logan.Entry) tss.KeyGenParty {
	switch protocol {
	case tss.ProtocolID_ECDSA_KEYGEN:
		return bnb.NewKeygenParty(self, parties, sessionId, logger)
	case tss.ProtocolID_FROST_KEYGEN:
		return frost.NewKeygenParty(self, parties, threshold, sessionId, logger)

	}

	return nil
}
