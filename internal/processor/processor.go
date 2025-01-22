package processor

import (
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/bridge"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
)

type Processor struct {
	//  TODO: Add Core connector

	clients Clientr
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
		return nil, errors.Wrap(err, "failed get deposit data")
	}

	// TODO: Add fetching withdrawal token data from Core

	//set deposit data to deposit structure
	depositAmount := depositData.DepositAmount.String()

	// TODO: This data has to be
	withdrawalAmount := "123"


//DepositIdentifier:  id,
//	DestinationChainId: eventBody.Network,
//		DestinationAddress: eventBody.Receiver,
//		TokenAddress:       bridge.DefaultNativeTokenAddress,
//		DepositAmount:      eventBody.Amount,
//		Block:              int64(log.BlockNumber),
//		SourceAddress:      from.String(),
	deposit := &database.Deposit{
		DepositIdentifier: dbIdentifier,
		Depositor:         &depositData.SourceAddress,
		DepositAmount:     &depositAmount,
		DepositToken:      &depositData.TokenAddress,
		Receiver:          &depositData.DestinationAddress,
		WithdrawalToken:   ,
		DepositBlock:      nil,
		WithdrawalStatus:  0,
		WithdrawalTxHash:  nil,
		WithdrawalChainId: nil,
		WithdrawalAmount:  nil,
		IsWrappedToken:    nil,
		Signature:         nil,
	}

	return

}
