package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	bnb "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frost "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"gitlab.com/distributed_lab/logan/v3"
)

func SelectSignByProtocol(protocol int, group curve.Curve, self tss.LocalSignParty, sessionId string, logger *logan.Entry) tss.SignParty {
	switch protocol {
	case tss.ProtocolID_ECDSA:
		return bnb.NewSignParty(self, sessionId, logger)
	case tss.ProtocolID_FROST:
		return frost.NewSignParty(self, sessionId, group, logger)
	}

	return nil
}

func SelectSignByShare(self tss.LocalSignParty, sessionId string, logger *logan.Entry) tss.SignParty {
	if self.FrostShare != nil {
		return SelectSignByProtocol(tss.ProtocolID_FROST, curve.Secp256k1{}, self, sessionId, logger)
	}

	return SelectSignByProtocol(tss.ProtocolID_ECDSA, curve.Secp256k1{}, self, sessionId, logger)
}
