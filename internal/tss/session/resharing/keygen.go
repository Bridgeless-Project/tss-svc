package resharing

import (
	"context"
	"time"

	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	tssKeygen "github.com/Bridgeless-Project/tss-svc/internal/tss/session/keygen"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ resharingTypes.Handler = &KeygenRound{}

type KeygenRound struct {
	parties        []p2p.Party
	sessionParams  session.Params
	secrets        secrets.Storage
	core           *coreConnector.Connector
	sessionManager *p2p.SessionManager

	logger *logan.Entry

	oldParty bool
	newParty bool
}

func NewKeygenRound() *KeygenRound {
	return &KeygenRound{}
}

func (r *KeygenRound) MaxHandleDuration() time.Duration {
	// includes time to
	// - init session delay (1 sec)
	// - run keygen protocol
	// - submit new pubkey to core
	// - listen for core confirmation
	return time.Second + 2*session.BoundaryKeygenSession
}

func (r *KeygenRound) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	// TODO: check if core has confirmed new pubkey and update state accordingly

	return false, nil
}

func (r *KeygenRound) Handle(ctx context.Context, state *resharingTypes.State) error {
	if r.oldParty && !r.newParty {
		// old party only - wait for confirmation of new pubkey
		return r.listenForConfirmation(ctx, state)
	}

	preparams, err := r.secrets.GetKeygenPreParams()
	if err != nil {
		return errors.Wrap(err, "failed to get preparams")
	}
	account, err := r.secrets.GetCoreAccount()
	if err != nil {
		return errors.Wrap(err, "failed to get core account")
	}

	keygenSession := tssKeygen.NewSession(
		tss.LocalKeygenParty{
			PreParams: *preparams,
			Address:   account.CosmosAddress(),
			// FIXME: calculate th dynamically
		},
		r.parties,
		r.sessionParams,
		r.logger,
	)
	r.sessionManager.Add(keygenSession)
	<-time.After(1 * time.Second) // slight delay to ensure session is registered before first message arrives

	if err = keygenSession.Run(ctx); err != nil {
		return errors.Wrap(err, "failed to start keygen session")
	}
	result, err := keygenSession.WaitFor()
	if err != nil {
		return errors.Wrap(err, "failed to produce key share")
	}

	if err = r.saveKeyShare(result); err != nil {
		return errors.Wrap(err, "failed to save key share")
	}

	// TODO: SUBMIT NEW TSS PUBKEY TO CORE

	if err = r.listenForConfirmation(ctx, state); err != nil {
		return errors.Wrap(err, "failed to listen for core confirmation")
	}

	return nil
}

func (r *KeygenRound) listenForConfirmation(ctx context.Context, state *resharingTypes.State) error {
	// TODO: listen for new pubkey confirmation from core and update state accordingly
	return nil
}

func (r *KeygenRound) saveKeyShare(result *keygen.LocalPartySaveData) error {
	var err error

	if r.newParty && !r.oldParty {
		err = r.secrets.SaveTssShare(result)
	} else {
		err = r.secrets.SaveTemporaryTssShare(result)
	}

	return err
}
