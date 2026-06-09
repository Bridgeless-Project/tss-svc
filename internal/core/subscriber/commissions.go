package subscriber

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	bridgeconfig "github.com/Bridgeless-Project/tss-svc/internal/bridge/config"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/commissions"
	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"gitlab.com/distributed_lab/logan/v3"
)

const opSubscriberCommission = "op-subscriber-commission"

type CommissionEventSubscriber struct {
	client            *http.HTTP
	connector         *connector.Connector
	subscriptionQuery string

	self           tss.LocalSignParty
	parties        []p2p.Party
	sessionManager *p2p.SessionManager

	bridgeSettings bridgeconfig.EvmSettings

	log *logan.Entry
}

func NewCommissionEventSubscriber(
	client *http.HTTP,
	connector *connector.Connector,
	self tss.LocalSignParty,
	parties []p2p.Party,
	sessionManager *p2p.SessionManager,
	bridgeSettings bridgeconfig.EvmSettings,
	logger *logan.Entry,
) *CommissionEventSubscriber {
	return &CommissionEventSubscriber{
		client:    client,
		connector: connector,
		subscriptionQuery: fmt.Sprintf(
			"tm.event='Tx' AND %s.%s EXISTS",
			bridgeTypes.EventType_DISTRIBUTE_FEES.String(),
			bridgeTypes.AttributeEpochId,
		),
		self:           self,
		parties:        parties,
		sessionManager: sessionManager,
		bridgeSettings: bridgeSettings,
		log:            logger,
	}
}

func (s *CommissionEventSubscriber) Run(ctx context.Context) error {
	out, err := s.client.Subscribe(ctx, opSubscriberCommission, s.subscriptionQuery, opPoolSize)
	if err != nil {
		return errors.Wrap(err, "subscriber init failed")
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("context cancelled, stopping receiving events")

			shutdownDeadline, cancel := context.WithTimeout(context.Background(), time.Second)
			err = s.client.Unsubscribe(shutdownDeadline, opSubscriberCommission, s.subscriptionQuery)
			cancel()

			return errors.Wrap(err, "failed to unsubscribe from events")
		case resultEvent, ok := <-out:
			if !ok {
				return errors.New("subscription channel closed")
			}

			s.log.Info("received event to process commission withdrawals")
			if count, err := s.processEvent(ctx, resultEvent); err != nil {
				s.log.WithError(err).Error("failed to process commission event")
			} else {
				s.log.WithField("withdrawals_count", count).Info("successfully processed commission event")
			}
		}
	}
}

func (s *CommissionEventSubscriber) processEvent(ctx context.Context, event coretypes.ResultEvent) (int, error) {
	var blockHeight int64
	switch data := event.Data.(type) {
	case types.EventDataTx:
		blockHeight = data.Height
	default:
		return 0, errors.Errorf("unexpected event data type: %T", event.Data)
	}

	data, err := parseCommissionEventData(blockHeight, event.Events)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse commission event data")
	}

	var blockTime time.Time
	if err = retry.Do(func() error {
		block, err := s.client.Block(ctx, &blockHeight)
		if err != nil {
			return err
		}
		blockTime = block.Block.Time

		return nil
	}, retry.Attempts(3), retry.Delay(3*time.Second)); err != nil {
		return 0, errors.Wrap(err, "failed to fetch block data for commission event")
	}

	withdrawals, err := commissions.NewSession(
		core.EventCommissionCollection{
			EventDataCommissionCollection: *data,
			Time:                          blockTime,
		},
		s.self,
		s.parties,
		s.connector,
		s.bridgeSettings,
		s.sessionManager,
		s.log.WithField("component", "commissions_withdrawal_sessions"),
	).Run(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to run commission withdrawal session")
	} else if len(withdrawals) == 0 {
		return 0, nil
	}

	if err = retry.Do(func() error {
		return s.connector.SubmitSystemWithdrawals(ctx, data.EpochId, withdrawals...)
	}, retry.Attempts(10), retry.Delay(3*time.Second)); err != nil {
		return 0, errors.Wrap(err, "failed to submit commission withdrawals")
	}

	return len(withdrawals), nil
}

func parseCommissionEventData(blockHeight int64, attributes map[string][]string) (*core.EventDataCommissionCollection, error) {
	for key, attribute := range attributes {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 || parts[0] != bridgeTypes.EventType_DISTRIBUTE_FEES.String() {
			continue
		}

		if parts[1] == bridgeTypes.AttributeEpochId {
			parsed, err := strconv.ParseUint(attribute[0], 10, 32)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse epoch id from attribute: %s", attribute[0])
			}

			return &core.EventDataCommissionCollection{
				EpochId:     uint32(parsed),
				BlockHeight: blockHeight,
			}, nil
		}
	}

	return nil, errors.New("event not found in attributes")
}
