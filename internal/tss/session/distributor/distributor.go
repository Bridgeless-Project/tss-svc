package distributor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/deposit"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

const DepositAcceptorSessionIdentifier = "DEPOSIT_DISTRIBUTION"

var _ p2p.TssSession = &DepositDistributionSession{}

type DistributedDepositMsg struct {
	Distributor core.Address
	Identifier  *types.DepositIdentifier
}

func (d *DistributedDepositMsg) GetIdentifier() db.DepositIdentifier {
	if d.Identifier == nil {
		return db.DepositIdentifier{}
	}

	return db.DepositIdentifier{
		ChainId: d.Identifier.ChainId,
		TxHash:  d.Identifier.TxHash,
		TxNonce: d.Identifier.TxNonce,
	}
}

type DepositDistributionSession struct {
	fetcher *deposit.Fetcher
	data    db.DepositsQ
	logger  *logan.Entry

	distributors map[core.Address]struct{}
	broadcaster  *broadcast.Broadcaster
	self         core.Address

	msgs       chan DistributedDepositMsg
	missingIds chan []db.DepositIdentifier
	jobs       chan db.DepositIdentifier
}

func NewDepositDistributionSession(
	self core.Address,
	distributors []p2p.Party,
	fetcher *deposit.Fetcher,
	data db.DepositsQ,
	logger *logan.Entry,
) *DepositDistributionSession {
	distributorsMap := make(map[core.Address]struct{}, len(distributors))
	for _, distributor := range distributors {
		distributorsMap[distributor.CoreAddress] = struct{}{}
	}

	return &DepositDistributionSession{
		fetcher:      fetcher,
		msgs:         make(chan DistributedDepositMsg, 1000),
		missingIds:   make(chan []db.DepositIdentifier, 1000),
		jobs:         make(chan db.DepositIdentifier, 2000),
		data:         data,
		logger:       logger,
		self:         self,
		broadcaster:  broadcast.NewBroadcaster(distributors, logger.WithField("component", "broadcaster")),
		distributors: distributorsMap,
	}
}

func (d *DepositDistributionSession) Run(ctx context.Context) {
	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		d.runDistributor(ctx)
		wg.Done()
	}()
	go func() {
		d.runAcceptor(ctx)
		wg.Done()
	}()

	wg.Wait()
}

func (d *DepositDistributionSession) runDistributor(ctx context.Context) {
	d.logger.Info("distributor started")

	var (
		statusPending = types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING
		cooldown      = time.Second * 0
	)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("distributor cancelled")
			return
		case <-time.After(cooldown):
			cooldown = time.Second * 5

			pendingDeposit, err := d.data.GetWithSelector(db.DepositsSelector{
				Status:         &statusPending,
				NotDistributed: true,
				One:            true,
			})
			if err != nil {
				d.logger.WithError(err).Error("failed to get pending deposit")
				continue
			} else if pendingDeposit == nil {
				continue
			}

			raw, _ := anypb.New(&p2p.DepositDistributionData{DepositId: pendingDeposit.ToMsgDepositIdentifier()})
			d.broadcaster.Broadcast(&p2p.SubmitRequest{
				Sender:    d.self.String(),
				Type:      p2p.RequestType_RT_DEPOSIT_DISTRIBUTION,
				SessionId: DepositAcceptorSessionIdentifier,
				Data:      raw,
			})

			if err = d.data.UpdateDistributedStatus(pendingDeposit.DepositIdentifier, true); err != nil {
				d.logger.
					WithField("deposit", pendingDeposit.DepositIdentifier.TxHash).
					WithError(err).
					Error("failed to update deposit as distributed")
				continue
			}

			cooldown = 500 * time.Millisecond
		}
	}
}

func (d *DepositDistributionSession) runAcceptor(ctx context.Context) {
	d.logger.Info("acceptor started")
	sem := make(chan struct{}, 20)

	go d.enqueueJobs(ctx)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("acceptor cancelled")
			return
		case id := <-d.jobs:
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				go func(id db.DepositIdentifier) {
					if err := d.processDeposit(id); err != nil {
						d.logger.WithError(err).Error("failed to process deposit")
					}
					<-sem
				}(id)
			}
		}
	}
}

func (d *DepositDistributionSession) Id() string {
	return DepositAcceptorSessionIdentifier
}

func (d *DepositDistributionSession) Receive(request *p2p.SubmitRequest) error {
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

	d.msgs <- DistributedDepositMsg{
		Distributor: sender,
		Identifier:  data.DepositId,
	}

	return nil
}

func (d *DepositDistributionSession) QueueMissing(ctx context.Context, ids []db.DepositIdentifier) {
	if len(ids) == 0 {
		return
	}
	select {
	case <-ctx.Done():
		d.logger.Info("context cancelled before send")
	case d.missingIds <- ids:
	}
}

func (d *DepositDistributionSession) processDeposit(id db.DepositIdentifier) error {
	log := d.logger.WithField("tx_hash", id.TxHash).WithField("chain_id", id.ChainId)
	deposit, err := d.data.Get(id)
	if err != nil {
		return errors.Wrap(err, "failed to check if deposit exists")
	} else if deposit != nil {
		log.Warnf("deposit already exists")
		return nil
	}

	deposit, err = d.fetcher.FetchDeposit(id)
	if err != nil {
		if chain.IsPendingDepositError(err) {
			log.Warn("deposit still pending")
			return nil
		}
		if chain.IsInvalidDepositError(err) || core.IsInvalidDepositError(err) {
			deposit = &db.Deposit{
				DepositIdentifier: id,
				WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_INVALID,
			}
			if _, err = d.data.Insert(*deposit); err != nil {
				return errors.Wrap(err, "failed to insert invalid deposit")
			}
		}
		return errors.Wrap(err, "failed to fetch deposit")
	}
	deposit.Distributed = true
	if _, err = d.data.Insert(*deposit); err != nil {
		if errors.Is(err, db.ErrAlreadySubmitted) {
			log.Info("deposit already found in db")
			return nil
		} else {
			return errors.Wrap(err, "failed to insert deposit")
		}
	}
	return nil
}

func (d *DepositDistributionSession) enqueueJobs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-d.msgs:
			d.jobs <- msg.GetIdentifier()
		case ids := <-d.missingIds:
			for _, id := range ids {
				d.jobs <- id
			}
		}
	}
}

// RegisterIdChangeListener is a no-op for DepositDistributionSession
func (d *DepositDistributionSession) RegisterIdChangeListener(func(oldId string, newId string)) {}

// SigningSessionInfo is a no-op for DepositDistributionSession
func (d *DepositDistributionSession) SigningSessionInfo() *p2p.SigningSessionInfo {
	return nil
}
