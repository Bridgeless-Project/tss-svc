package syncer

import (
	"context"
	"fmt"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"time"
)

type Syncer struct {
	maxRetires int
	ctx        context.Context

	connection *grpc.ClientConn
}

func NewSyncer(maxRetires int, ctx context.Context) *Syncer {
	return &Syncer{
		maxRetires: maxRetires,
		ctx:        ctx,
	}
}

func (s *Syncer) Sync(chainId string) (info *p2p.SessionInfo, err error) {
	if s.connection == nil {
		return nil, errors.New("connection not initialized")
	}
	// validate whether provided party is eligible to sync with
	status, err := p2p.NewP2PClient(s.connection).Status(s.ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to party")
	}
	if status == nil {
		return nil, errors.New("got nil status")
	}
	if status.Status != p2p.PartyStatus_PS_SIGN {
		return nil, errors.New(fmt.Sprintf("invalid party status: %v", status))
	}

	for i := 0; i < s.maxRetires; i++ {
		info, err = p2p.NewP2PClient(s.connection).GetSessionInfo(s.ctx, &p2p.SessionInfoRequest{
			ChainId: chainId,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get session info")
		}
		if info.NextSessionStartTime != 0 {
			// if next session start time is received party is ready to start session
			break
		}

		time.Sleep(time.Second * time.Duration(info.NextSessionStartTime/5))
	}

	return info, nil
}

func (s *Syncer) FindPartyToSync(parties []p2p.Party) (*grpc.ClientConn, error) {
	for _, party := range parties {
		status, err := p2p.NewP2PClient(party.Connection()).Status(s.ctx, nil)
		if err != nil {

			continue
		}
		if status.GetStatus() == p2p.PartyStatus_PS_SIGN {
			return party.Connection(), nil
		}
	}

	return nil, errors.New("no party to sync")
}

func (s *Syncer) WithParty(conn *grpc.ClientConn) *Syncer {
	s.connection = conn
	return s
}
