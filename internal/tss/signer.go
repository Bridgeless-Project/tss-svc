package tss

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/bnb-chain/tss-lib/v3/common"
)

type LocalSignParty struct {
	Account   core.Account
	Share     Share
	Threshold int
}

type SignParty interface {
	WithParties(parties []p2p.Party) SignParty
	WithSigningData(data []byte) SignParty
	Run(ctx context.Context)
	WaitFor() *common.SignatureData
	Receive(sender core.Address, data *p2p.TssData)
}
