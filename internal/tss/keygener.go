package tss

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
)

type KeyGenParty interface {
	Run(ctx context.Context)
	WaitFor() Share
	Receive(sender core.Address, data *p2p.TssData)
}
