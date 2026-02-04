package subscriber

import (
	"time"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	"golang.org/x/net/context"
)

type KeygenEvent struct {
	// Define fields for the KeygenEvent as per requirements
}

type KeygenEventSubscriber struct {
	client *http.HTTP
	log    *logan.Entry
}

func NewKeygenEventSubscriber(client *http.HTTP, logger *logan.Entry) *KeygenEventSubscriber {
	return &KeygenEventSubscriber{
		client: client,
		log:    logger,
	}
}

func (s *KeygenEventSubscriber) Run(ctx context.Context) (*KeygenEvent, error) {
	query := "tm.event='Tx'" //FIXME: complete query when event is defined
	out, err := s.client.Subscribe(ctx, "", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe to keygen events")
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("keygen event subscriber stopped")
			shutdownDeadline, cancel := context.WithTimeout(context.Background(), time.Second)
			if err = s.client.Unsubscribe(shutdownDeadline, "OpServiceName", query); err != nil {
				s.log.WithError(err).Error("failed to unsubscribe from keygen events")
			}

			cancel()
			return nil
		}
	}
}
