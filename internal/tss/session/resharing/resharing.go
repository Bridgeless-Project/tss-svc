package resharing

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"golang.org/x/sync/errgroup"
)

type Session struct {
	params session.Params

	oldSet []p2p.Party
	newSet []p2p.Party

	selfOld, selfNew bool

	secrets        secrets.Storage
	sessionManager *p2p.SessionManager

	logger *logan.Entry
}

func NewSession(
	params session.Params,
	oldSet, newSet []p2p.Party,
	secrets secrets.Storage,
	sessionManager *p2p.SessionManager,
) *Session {
	return &Session{
		params: params,
		oldSet: oldSet,
		newSet: newSet,
	}
}

// old set - wait for pk; sign with old share; done
// new set - keygen; save new share; done
// continuers - keygen; wait for pk; sing with old share; save new share; done

func (s *Session) Run(ctx context.Context) error {
	// TODO: create base handler for keygen and confirmation event listener
	// TODO: for chain in chains:
	// - handlers for btc sessions
	// - handler for zano session
	// - handler for evm session
	// - handler for solana session
	// - handler for ton session
	// TODO: create handler for swapping shares

	state := resharingTypes.InitializeState(s.params.StartTime)

	keygenRound := NewKeygenRound()
	maxKeygenDuration := keygenRound.MaxHandleDuration()
	keygenManager := resharingTypes.NewHandlerManager(
		keygenRound, state, s.logger.WithField("component", "resharing_keygen_manager"),
	)

	if err := keygenManager.Manage(ctx, s.params.StartTime); err != nil {
		return errors.Wrap(err, "failed to manage keygen round")
	}

	if s.selfNew && !s.selfOld {
		// new party only - no further steps
		return nil
	}

	sessionStartTime := s.params.StartTime.Add(maxKeygenDuration)
	state.SessionStartTime = sessionStartTime

	managers := []resharingTypes.HandlerManager{}
	// TODO: configure managers

	eg, egCtx := errgroup.WithContext(ctx)
	for _, manager := range managers {
		eg.Go(func() error {
			return manager.Manage(egCtx, sessionStartTime)
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "failed to run resharing session handlers")
	}

	// TODO: SAVE STATE ON CORE

	// TODO: IMPORT WALLETS FOR UTXO networks (all parties)

	if s.selfOld && !s.selfNew {
		// old party only - no further steps
		return nil
	}

	// TODO: SAVE NEW SHARE IF CONTINUER

	return nil
}
