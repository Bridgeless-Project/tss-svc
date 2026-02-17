package resharing

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
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
		return r.listenForPubkeyConfirmation(ctx, state)
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
			Threshold: r.sessionParams.Threshold,
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

	pubkey := bridge.PubkeyToString(result.ECDSAPub.X(), result.ECDSAPub.Y())

	var maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		err = r.core.SetEpochPubKey(state.Epoch, pubkey)
		if err != nil {
			r.logger.WithError(err).Errorf("failed to submit new pubkey to core (attempt %d/%d)", i+1, maxRetries)
			continue
		}

		if err = r.listenForPubkeyConfirmation(ctx, state); err != nil {
			return errors.Wrap(err, "failed to confirm new pubkey on core")
		}
	}

	return errors.New("failed to submit new pubkey to core after maximum retries")
}

func (r *KeygenRound) listenForPubkeyConfirmation(ctx context.Context, state *resharingTypes.State) error {
	const cooldown = 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return errors.New("context cancelled while waiting for pubkey confirmation")
		case <-time.After(cooldown):
			pubkey, err := r.core.GetEpochPubKey(state.Epoch)
			if err != nil {
				if errors.Is(err, core.ErrEpochNotFound) {
					// epoch not found means core hasn't processed the new pubkey yet, so we just wait and retry
					continue
				}

				r.logger.WithError(err).Error("failed to get epoch pubkey from core, will retry...")
			}

			ecdsaPub, err := bridge.DecodePubkey(pubkey)
			if err != nil {
				return errors.Wrap(err, "failed to decode received pubkey from core")
			}

			state.NewPubKey = ecdsaPub
			return nil
		}
	}
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
