package tss

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
)

type KeyGenParty interface {
	Run(ctx context.Context)
	WaitFor() *LocalPartyData
	Receive(sender core.Address, data *p2p.TssData)
	//receiveMsgs(ctx context.Context)
	//receiveUpdates(ctx context.Context, out <-chan tss.Message, end <-chan *LocalPartyData)
}

func NewLocalPartyData(data interface{}) *LocalPartyData {
	return &LocalPartyData{
		data: data,
	}
}

type LocalPartyData struct {
	data interface{}
}
