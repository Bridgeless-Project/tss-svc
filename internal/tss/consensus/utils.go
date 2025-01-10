package consensus

import (
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
)

type partyMsg struct {
	Type        p2p.RequestType
	Sender      core.Address
	WireMsg     []byte
	IsBroadcast bool
}
