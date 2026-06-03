package subscriber

import (
	"context"
	"strconv"
	"strings"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	"github.com/tendermint/tendermint/types"
	"gitlab.com/distributed_lab/logan/v3"
)

const opSubscriberCommission = "op-subscriber-commission"

type CommissionEventSubscriber struct {
	client            *http.HTTP
	subscriptionQuery string

	log *logan.Entry
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

			var blockHeight int64
			switch data := resultEvent.Data.(type) {
			case types.EventDataTx:
				blockHeight = data.Height
			default:
				s.log.Warnf("unexpected event data type: %T", resultEvent.Data)
				continue
			}

			data, err := parseCommissionEventData(blockHeight, resultEvent.Events)
			if err != nil {
				s.log.Warnf("failed to parse commission event data: %v", err)
				continue
			}

			s.log.Infof("received commission event for epoch %d at block height %d", data.EpochId, data.BlockHeight)
		}
	}
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
