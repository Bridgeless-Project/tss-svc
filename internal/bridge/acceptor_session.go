package bridge

import (
	"context"
	"fmt"

	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ p2p.TssSession = &DepositAcceptorSession{}

const (
	DepositAcceptorSessionIdentifier = "DEPOSIT_DISTRIBUTION"
)

type distributedDeposit struct {
	Distributor core.Address
	Identifier  *types.DepositIdentifier
}

type DepositAcceptorSession struct {
	fetcher *DepositFetcher
	data    db.DepositsQ
	logger  *logan.Entry
	clients bridgeTypes.ClientsRepository

	distributors map[core.Address]struct{}

	msgs chan distributedDeposit
}

func NewDepositAcceptorSession(
	distributors []p2p.Party,
	fetcher *DepositFetcher,
	clients bridgeTypes.ClientsRepository,
	data db.DepositsQ,
	logger *logan.Entry,
) *DepositAcceptorSession {
	distributorsMap := make(map[core.Address]struct{}, len(distributors))
	for _, distributor := range distributors {
		distributorsMap[distributor.CoreAddress] = struct{}{}
	}

	return &DepositAcceptorSession{
		fetcher:      fetcher,
		msgs:         make(chan distributedDeposit, 100),
		data:         data,
		logger:       logger,
		clients:      clients,
		distributors: distributorsMap,
	}
}

func (d *DepositAcceptorSession) Run(ctx context.Context) {
	d.logger.Info("session started")

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("session cancelled")
			return
		case msg := <-d.msgs:
			d.logger.Info(fmt.Sprintf("received deposit from %s", msg.Distributor))

			client, err := d.clients.Client(msg.Identifier.ChainId)
			if err != nil {
				d.logger.Error("got unsupported chain identifier")
				continue
			}

			if exists, err := d.data.Exists(db.ToExistenceCheck(msg.Identifier, client.Type())); err != nil {
				d.logger.WithError(err).Error("failed to check if deposit exists")
				continue
			} else if exists {
				d.logger.Info("deposit already exists")
				continue
			}

			deposit, err := d.fetcher.FetchDeposit(db.DepositIdentifier{
				ChainId: msg.Identifier.ChainId,
				TxHash:  msg.Identifier.TxHash,
				TxNonce: int(msg.Identifier.TxNonce),
			})
			if err != nil {
				// TODO: checkout err type
				d.logger.WithError(err).Error("failed to fetch deposit")
				continue
			}

			if _, err = d.data.Insert(*deposit); err != nil {
				d.logger.WithError(err).Error("failed to insert deposit")
				continue
			}

			d.logger.Info("deposit successfully fetched")
		}
	}
}

func (d *DepositAcceptorSession) Id() string {
	return DepositAcceptorSessionIdentifier
}

func (d *DepositAcceptorSession) Receive(request *p2p.SubmitRequest) error {
	if request == nil || request.Data == nil {
		return errors.New("nil request")
	}
	if request.Type != p2p.RequestType_RT_DEPOSIT_DISTRIBUTION {
		return errors.New("invalid request type")
	}
	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}

	if _, ok := d.distributors[sender]; !ok {
		return errors.New(fmt.Sprintf("sender '%s' is not a valid deposit distributor", sender))
	}

	data := &p2p.DepositDistributionData{}
	if err = request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal deposit identifier")
	}
	if data == nil || data.DepositId == nil {
		return errors.New("nil deposit identifier")
	}

	d.msgs <- distributedDeposit{
		Distributor: sender,
		Identifier:  data.DepositId,
	}

	return nil
}

// RegisterIdChangeListener is a no-op for DepositAcceptorSession
func (d *DepositAcceptorSession) RegisterIdChangeListener(func(oldId string, newId string)) {
	return
}
