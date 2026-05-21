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
}

func NewLocalPartyData(data interface{}) *LocalPartyData {
	return &LocalPartyData{
		data: data,
	}
}

type LocalPartyData struct {
	data interface{}
}

func (d *LocalPartyData) GetData() interface{} {
	if d == nil {
		return nil
	}

	return d.data
}
