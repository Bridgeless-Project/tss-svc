package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	bnb "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frost "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"gitlab.com/distributed_lab/logan/v3"
)

func SelectSignByProtocol(self tss.LocalSignParty, sessionId string, logger *logan.Entry) tss.SignParty {
	switch self.Share.Protocol() {
	case tss.ProtocolID_FROST:
		return frost.NewSignParty(self, sessionId, logger)
	case tss.ProtocolID_ECDSA:
		return bnb.NewSignParty(self, sessionId, logger)
	}

	return nil
}
