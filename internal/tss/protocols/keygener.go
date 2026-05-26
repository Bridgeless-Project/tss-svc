package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	bnb "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frost "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"gitlab.com/distributed_lab/logan/v3"
)

func SelectKeyGenByProtocol(self tss.LocalKeygenParty, parties []p2p.Party, sessionId string, logger *logan.Entry) tss.KeyGenParty {
	switch self.PreParams.Protocol() {
	case "ecdsa":
		return bnb.NewKeygenParty(self, parties, sessionId, logger)
	case "frost":
		return frost.NewKeygenParty(self, parties, sessionId, logger)
	}

	return nil
}
