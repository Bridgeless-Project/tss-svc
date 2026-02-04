package types

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Handler interface {
	Handle(context.Context, *State) error
	MaxHandleDuration() time.Duration
	RecoverStateIfProcessed(*State) (bool, error)
}

type HandlerManager struct {
	handler Handler
	state   *State
	logger  *logan.Entry
}

func NewHandlerManager(
	handler Handler,
	state *State,
	logger *logan.Entry,
) *HandlerManager {
	return &HandlerManager{
		handler: handler,
		state:   state,
		logger:  logger,
	}
}

func (m *HandlerManager) Manage(ctx context.Context, startTime time.Time) error {
	if processed, err := m.handler.RecoverStateIfProcessed(m.state); err != nil {
		return errors.Wrap(err, "failed to check if handler is already processed")
	} else if processed {
		return nil
	}

	maxDelay := m.handler.MaxHandleDuration()
	runDelay := time.Until(startTime)

	m.logger.Infof("handler will start in %s", runDelay)
	select {
	case <-ctx.Done():
		m.logger.Info("handler cancelled")
		return nil
	case <-time.After(runDelay):
		m.logger.Info("handler started")
	}

	handleCtx, cancel := context.WithTimeout(ctx, maxDelay)
	defer cancel()

	if err := m.handler.Handle(handleCtx, m.state); err != nil {
		return errors.Wrap(err, "failed to execute handler")
	}

	return nil
}
