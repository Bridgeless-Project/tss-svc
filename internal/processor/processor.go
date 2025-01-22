package processor

import (
	"github.com/hyle-team/tss-svc/internal/api/common"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
)

type Processor struct {
	//  TODO: Add Core connector

	client bridgeTypes.Client
	db     database.DepositsQ
}

func NewProcessor(client bridgeTypes.Client, db database.DepositsQ) *Processor {
	return &Processor{
		client: client,
		db:     db,
	}
}

func (p *Processor) FetchDepositData(identifier *types.DepositIdentifier) (*database.Deposit, error) {
	// form db identifier

	dbIdentifier := common.FormDepositIdentifier(identifier, p.client.Type())

	//get deposit data from network
	depositData, err := p.client.GetDepositData(dbIdentifier)

	if err != nil {
		return nil, errors.Wrap(err, "get deposit data")
	}
	return

}
