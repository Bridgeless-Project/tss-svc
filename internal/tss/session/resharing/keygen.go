package resharing

import (
	"context"
	"fmt"
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
	"github.com/avast/retry-go"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ resharingTypes.Handler = &KeygenHandler{}

type KeygenHandler struct {
	parties        []p2p.Party
	secrets        secrets.Storage
	core           *coreConnector.Connector
	sessionManager *p2p.SessionManager

	logger *logan.Entry

	oldEpochMember, newEpochMember bool
}

func NewKeygenHandler(
	parties []p2p.Party,
	secrets secrets.Storage,
	core *coreConnector.Connector,
	sessionManager *p2p.SessionManager,
	logger *logan.Entry,
	oldEpochMember, newEpochMember bool,
) *KeygenHandler {
	return &KeygenHandler{
		parties:        parties,
		secrets:        secrets,
		core:           core,
		sessionManager: sessionManager,
		logger:         logger.WithField("component", "resharing_keygen_handler"),
		oldEpochMember: oldEpochMember,
		newEpochMember: newEpochMember,
	}
}

func (r *KeygenHandler) MaxHandleDuration() time.Duration {
	// includes time to
	// - init session delay (1 sec)
	// - run keygen protocol
	// - submit new pubkey to core
	// - listen for core confirmation
	return time.Second + 2*session.BoundaryKeygenSession
}

func (r *KeygenHandler) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	var (
		pubkey string
		err    error
	)

	err = retry.Do(
		func() error {
			pubkey, err = r.core.GetEpochPubKey(state.Epoch)
			if errors.Is(err, core.ErrEpochNotFound) {
				return nil // pubkey not set yet, not an error
			}

			return err
		},
		retry.Attempts(3),
		retry.Delay(5*time.Second),
	)
	if err != nil {
		return false, errors.Wrap(err, "failed to get epoch pubkey from core")
	}
	if pubkey == "" {
		return false, nil // pubkey not set yet
	}

	ecdsaPub, err := bridge.DecodePubkey(pubkey)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode received pubkey from core")
	}

	state.NewPubKey = ecdsaPub

	if r.oldEpochMember && !r.newEpochMember {
		return true, nil
	}

	// should load new share from temporary secrets
	if r.oldEpochMember {
		state.NewShare, err = r.secrets.GetTemporaryTssShare()
	} else {

		// TODO: rewrite for frost
		var share interface{}
		share, _, err = r.secrets.GetTssShare()
		state.NewShare = share.(*keygen.LocalPartySaveData)
	}
	if err != nil {
		return false, errors.Wrap(err, "failed to get key share from secrets storage")
	}

	if !state.NewPubKey.Equal(state.NewShare.ECDSAPub.ToECDSAPubKey()) {
		return false, errors.New(fmt.Sprintf(
			"pubkey from core does not match pubkey derived from saved share: %s vs %s",
			pubkey,
			bridge.PubkeyPrefixedToString(state.NewShare.ECDSAPub.X(), state.NewShare.ECDSAPub.Y()),
		))
	}

	return true, nil
}

func (r *KeygenHandler) Handle(ctx context.Context, state *resharingTypes.State) error {
	if r.oldEpochMember && !r.newEpochMember {
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
			Threshold: int(state.Threshold),
		},
		r.parties,
		session.Params{Id: int64(state.Epoch)},
		r.logger,
		tss.ProtocolID_ECDSA,
		nil, // don't need to set the curve for ECDSA
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

	state.NewShare = result.GetData().(*keygen.LocalPartySaveData)
	if err = r.saveKeyShare(state.NewShare); err != nil {
		return errors.Wrap(err, "failed to save key share")
	}

	pubkey := bridge.PubkeyPrefixedToString(state.NewShare.ECDSAPub.X(), state.NewShare.ECDSAPub.Y())

	err = retry.Do(
		func() error { return r.core.SetEpochPubKey(state.Epoch, pubkey) },
		retry.Attempts(3),
		retry.Delay(5*time.Second),
	)
	if err != nil {
		return errors.Wrap(err, "failed to submit new pubkey to core")
	}

	if err = r.listenForPubkeyConfirmation(ctx, state); err != nil {
		return errors.Wrap(err, "failed to confirm new pubkey on core")
	}

	return nil
}

func (r *KeygenHandler) listenForPubkeyConfirmation(ctx context.Context, state *resharingTypes.State) error {
	r.logger.Debug("listening for new pubkey confirmation from core...")

	cooldown := 0 * time.Second
	for {
		select {
		case <-ctx.Done():
			return errors.New("context cancelled while waiting for pubkey confirmation")
		case <-time.After(cooldown):
			cooldown = 5 * time.Second

			pubkey, err := r.core.GetEpochPubKey(state.Epoch)
			if err != nil {
				if !errors.Is(err, core.ErrEpochNotFound) {
					r.logger.WithError(err).Error("failed to get epoch pubkey from core, will retry...")
				}

				r.logger.Info("new pubkey not yet confirmed on core, will retry...")

				continue
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

func (r *KeygenHandler) saveKeyShare(result *keygen.LocalPartySaveData) error {
	r.logger.Debug("saving new key share to secrets storage...")

	if r.oldEpochMember {
		return errors.Wrap(r.secrets.SaveTssShare(secrets.TssShareKeyTemporary, result), "failed to save temporary key share")
	}

	return errors.Wrap(r.secrets.SaveTssShare(secrets.TssShareKeyECDSA, result), "failed to save key share")
}
