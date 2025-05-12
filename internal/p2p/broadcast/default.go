package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/hyle-team/tss-svc/internal/p2p"
	"gitlab.com/distributed_lab/logan/v3"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Broadcaster struct {
	parties map[core.Address]p2p.Party

	logger *logan.Entry
}

func NewBroadcaster(to []p2p.Party, logger *logan.Entry) *Broadcaster {
	b := &Broadcaster{
		parties: make(map[core.Address]p2p.Party, len(to)),
		logger:  logger,
	}

	for _, party := range to {
		b.parties[party.CoreAddress] = party
	}

	return b
}

func (b *Broadcaster) Send(msg *p2p.SubmitRequest, to core.Address) error {
	party, exists := b.parties[to]
	if !exists {
		return errors.New("party not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), p2p.DefaultConnectionTimeout)
	defer cancel()

	if err := b.send(ctx, msg, party.Connection()); err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return nil
}

func (b *Broadcaster) send(ctx context.Context, msg *p2p.SubmitRequest, conn *grpc.ClientConn) error {
	_, err := p2p.NewP2PClient(conn).Submit(ctx, msg)

	return err
}

func (b *Broadcaster) Broadcast(msg *p2p.SubmitRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), p2p.DefaultConnectionTimeout+time.Second)

	wg := sync.WaitGroup{}
	wg.Add(len(b.parties))

	go func() { wg.Wait(); cancel() }()
	for _, party := range b.parties {
		go func(p p2p.Party) {
			defer wg.Done()
			if err := b.send(ctx, msg, p.Connection()); err != nil {
				b.logger.WithFields(logan.F{
					"receiver":   p.CoreAddress,
					"session_id": msg.SessionId,
				}).Debugf("failed to broadcast message: %s", err)
			}
		}(party)
	}
}
