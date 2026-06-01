package tss

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
)

type LocalSignParty struct {
	Account       core.Account
	Share         Share
	Threshold     int
	SignatureData SignatureData // TODO: maybe remove
}

type SignParty interface {
	WithParties(parties []p2p.Party) SignParty
	WithSigningData(data []byte) SignParty
	Run(ctx context.Context)
	WaitFor() SignatureData
	Receive(sender core.Address, data *p2p.TssData)
}
