package acceptor

import (
	"context"
	"fmt"

	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/bridge/deposit"
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
	fetcher *deposit.Fetcher
	data    db.DepositsQ
	logger  *logan.Entry

	distributors map[core.Address]struct{}

	msgs chan distributedDeposit
}

func NewDepositAcceptorSession(
	distributors []p2p.Party,
	fetcher *deposit.Fetcher,
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

			id := db.DepositIdentifier{
				ChainId: msg.Identifier.ChainId,
				TxHash:  msg.Identifier.TxHash,
				TxNonce: int(msg.Identifier.TxNonce),
			}

			deposit, err := d.data.Get(db.DepositIdentifier{
				TxHash:  msg.Identifier.TxHash,
				TxNonce: int(msg.Identifier.TxNonce),
				ChainId: msg.Identifier.ChainId,
			})
			if err != nil {
				d.logger.WithError(err).Error("failed to check if deposit exists")
				continue
			} else if deposit != nil {
				d.logger.Warn("deposit already exists")
				continue
			}

			deposit, err = d.fetcher.FetchDeposit(id)
			if err != nil {
				if chain.IsPendingDepositError(err) {
					d.logger.Warn("deposit still pending")
					continue
				}
				if chain.IsInvalidDepositError(err) || core.IsInvalidDepositError(err) {
					deposit = &db.Deposit{
						DepositIdentifier: id,
						WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_INVALID,
					}
					if _, err = d.data.Insert(*deposit); err != nil {
						d.logger.WithError(err).Error("failed to process deposit")
						continue
					}
					d.logger.Warn("invalid deposit")
					continue
				}
				d.logger.WithError(err).Error("failed to fetch deposit")
				continue
			}

			if _, err = d.data.Insert(*deposit); err != nil {
				if errors.Is(err, db.ErrAlreadySubmitted) {
					d.logger.Info("deposit already found in db")
				} else {
					d.logger.WithError(err).Error("failed to insert deposit")
				}
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

// SigningSessionInfo is a no-op for DepositAcceptorSession
func (d *DepositAcceptorSession) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
