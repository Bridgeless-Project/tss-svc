package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

const DefaultSyncRetriesCount = 5

type Syncer struct {
	maxRetries     int
	requiredStatus PartyStatus
	aliveParty     *Party
}

func NewSyncer(parties []Party, requiredStatus PartyStatus) (*Syncer, error) {
	aliveParty := findAliveParty(parties, requiredStatus)
	if aliveParty == nil {
		return nil, errors.New("no alive party found")
	}

	return &Syncer{
		requiredStatus: requiredStatus,
		maxRetries:     DefaultSyncRetriesCount,
		aliveParty:     aliveParty,
	}, nil
}

func (s *Syncer) Sync(ctx context.Context, chainId string) (info *SigningSessionInfo, err error) {
	if s.aliveParty == nil {
		return nil, errors.New("no alive party specified")
	}

	client := NewP2PClient(s.aliveParty.Connection())
	request := &SigningSessionInfoRequest{ChainId: chainId}

	for i := 0; i < s.maxRetries; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		info, err = client.GetSigningSessionInfo(timeoutCtx, request)
		cancel()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get session info")
		}
		if info.NextSessionStartTime != 0 {
			return info, nil
		}

		time.Sleep(time.Second * 1)
	}

	return nil, errors.New(fmt.Sprintf("no response from party %s after %d retries", s.aliveParty.CoreAddress, s.maxRetries))
}

func findAliveParty(parties []Party, requiredStatus PartyStatus) *Party {
	for _, party := range parties {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
		status, err := NewP2PClient(party.Connection()).Status(timeoutCtx, nil)
		cancel()
		if err != nil {
			continue
		}
		if status.GetStatus() == requiredStatus {
			return &party
		}
	}

	return nil
}
