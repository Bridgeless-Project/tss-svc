package p2p

import (
	"context"
	"time"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const DefaultConnectionTimeout = time.Second

type Broadcaster struct {
	parties map[core.Address]Party
}

func NewBroadcaster(to []Party) *Broadcaster {
	b := &Broadcaster{
		parties: make(map[core.Address]Party, len(to)),
	}

	for _, party := range to {
		b.parties[party.CoreAddress] = party
	}

	return b
}

func (b *Broadcaster) Send(msg *SubmitRequest, to core.Address) error {
	party, exists := b.parties[to]
	if !exists {
		return errors.New("party not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
	defer cancel()

	if err := b.send(ctx, msg, party.Connection()); err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return nil
}

func (b *Broadcaster) send(ctx context.Context, msg *SubmitRequest, conn *grpc.ClientConn) error {
	_, err := NewP2PClient(conn).Submit(ctx, msg)

	return err
}

func (b *Broadcaster) Broadcast(msg *SubmitRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout+time.Second)
	defer cancel()

	errGroup, errCtx := errgroup.WithContext(ctx)

	for _, party := range b.parties {
		errGroup.Go(func() error {
			return b.send(errCtx, msg, party.Connection())
		})
	}

	return errGroup.Wait()
}
