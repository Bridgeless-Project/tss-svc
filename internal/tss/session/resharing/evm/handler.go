package evm

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Handler struct {
	self           tss.LocalSignParty
	parties        []p2p.Party
	sessionManager *p2p.SessionManager

	logger *logan.Entry
}

func NewHandler(
	self tss.LocalSignParty,
	parties []p2p.Party,
	sessionManager *p2p.SessionManager,
	logger *logan.Entry,
) *Handler {
	return &Handler{
		self:           self,
		parties:        parties,
		sessionManager: sessionManager,
		logger:         logger,
	}
}

func (r *Handler) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	// TODO: check if core has confirmed resharing state and update state accordingly

	return false, nil
}

func (r *Handler) MaxHandleDuration() time.Duration {
	// includes time to perform two signing operations and session init delays
	return 2*session.BoundarySign + 2*time.Second
}

func (r *Handler) Handle(ctx context.Context, state *resharingTypes.State) error {
	var (
		newSigner             = evm.PubkeyToAddress(state.NewPubKey.X, state.NewPubKey.Y)
		oldSigner             = evm.PubkeyToAddress(r.self.Share.ECDSAPub.X(), r.self.Share.ECDSAPub.Y())
		addSignerOperation    = operations.NewAddSignerOperation(newSigner, state.GlobalStartTime)
		removeSignerOperation = operations.NewRemoveSignerOperation(oldSigner, state.GlobalStartTime)
	)

	addSignerSession := signing.NewSession(
		r.self,
		signing.SessionParams{
			Params: session.Params{
				Threshold: r.self.Threshold,
			},
			SigningData: addSignerOperation.CalculateHash(),
		},
		r.parties,
		r.logger.WithField("component", "resharing_evm_add_signer"),
	)
	r.sessionManager.Add(addSignerSession)
	<-time.After(time.Second) // // slight delay to ensure session is registered before first message arrives

	if err := addSignerSession.Run(ctx); err != nil {
		return errors.Wrap(err, "failed to run add signer session")
	}
	addSignerResult, err := addSignerSession.WaitFor()
	if err != nil {
		return errors.Wrap(err, "failed to produce add signer signature")
	}

	removeSignerSession := signing.NewSession(
		r.self,
		signing.SessionParams{
			Params: session.Params{
				Threshold: r.self.Threshold,
			},
			SigningData: removeSignerOperation.CalculateHash(),
		},
		r.parties,
		r.logger.WithField("component", "resharing_evm_remove_signer"),
	)
	r.sessionManager.Add(removeSignerSession)
	<-time.After(time.Second) // // slight delay to ensure session is registered before first message arrives

	if err = removeSignerSession.Run(ctx); err != nil {
		return errors.Wrap(err, "failed to run remove signer session")
	}
	removeSignerResult, err := removeSignerSession.WaitFor()
	if err != nil {
		return errors.Wrap(err, "failed to produce remove signer signature")
	}

	state.EvmData.AddNewSignerSignature = resharingTypes.UpdateSignerEvmSignature{
		Signer:    addSignerOperation.Signer.String(),
		StartTime: addSignerOperation.StartTime,
		Deadline:  addSignerOperation.Deadline,
		Nonce:     addSignerOperation.Nonce.Uint64(),
		Signature: evm.ConvertSignature(addSignerResult),
	}
	state.EvmData.RemoveOldSignerSignature = resharingTypes.UpdateSignerEvmSignature{
		Signer:    removeSignerOperation.Signer.String(),
		StartTime: removeSignerOperation.StartTime,
		Deadline:  removeSignerOperation.Deadline,
		Nonce:     removeSignerOperation.Nonce.Uint64(),
		Signature: evm.ConvertSignature(removeSignerResult),
	}

	return nil
}
