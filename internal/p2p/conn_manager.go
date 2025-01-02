package p2p

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/hyle-team/tss-svc/internal/core"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/grpc"
)

type connection struct {
	conn   *grpc.ClientConn
	status PartyStatus
}

type ConnectionManager struct {
	conns  map[core.Address]connection
	connsM *sync.RWMutex

	ready          map[core.Address]struct{}
	readyM         *sync.RWMutex
	requiredStatus PartyStatus

	subscribers map[chan struct{}]int
	subM        *sync.RWMutex
	logger      *logan.Entry
}

func NewConnectionManager(parties []Party, requiredStatus PartyStatus, logger *logan.Entry) *ConnectionManager {
	conns := make(map[core.Address]connection, len(parties))

	for _, p := range parties {
		conns[p.CoreAddress] = connection{conn: p.Connection(), status: PartyStatus_UNKNOWN}
	}

	return &ConnectionManager{
		conns:          conns,
		ready:          make(map[core.Address]struct{}, len(parties)),
		requiredStatus: requiredStatus,
		subscribers:    make(map[chan struct{}]int),
		logger:         logger,
	}
}

func (c *ConnectionManager) Subscribe(readyPartiesCount int) chan struct{} {
	c.subM.Lock()
	defer c.subM.Unlock()

	ch := make(chan struct{}, 1)
	c.subscribers[ch] = readyPartiesCount

	return ch
}

func (c *ConnectionManager) GetReady() map[core.Address]struct{} {
	c.readyM.RLock()
	defer c.readyM.RUnlock()

	return maps.Clone(c.ready)
}

func (c *ConnectionManager) watchStatuses() {
	ticker := time.NewTicker(10 * time.Second)
	keys := make([]core.Address, 0, len(c.conns))

	c.connsM.RLock()
	for k := range c.conns {
		keys = append(keys, k)
	}
	c.connsM.RUnlock()

	for ; ; <-ticker.C {
		for _, k := range keys {
			c.connsM.Lock()
			conn := c.conns[k]

			ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
			response, err := NewP2PClient(conn.conn).Status(ctx, nil)
			cancel()
			if err != nil {
				c.logger.WithError(err).WithField("party", k).Debug("Failed to get peer status")
			}

			responseStatus := response.GetStatus()
			// if status is the same, do nothing
			if conn.status == responseStatus {
				c.connsM.Unlock()
				continue
			}
			// when new status is required one, update ready list
			if responseStatus == c.requiredStatus {
				c.readyM.Lock()
				c.ready[k] = struct{}{}
				c.readyM.Unlock()

				c.subM.Lock()
				for ch, count := range c.subscribers {
					if len(c.ready) >= count {
						ch <- struct{}{}
						delete(c.subscribers, ch)
					}
				}
				c.subM.Unlock()
			}
			// when old status was required one, remove from ready list
			if conn.status == c.requiredStatus {
				c.readyM.Lock()
				delete(c.ready, k)
				c.readyM.Unlock()
			}

			conn.status = responseStatus
			c.conns[k] = conn
			c.connsM.Unlock()
		}
	}
}
