package resharing

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type UpdateContractHandler struct {
	self           tss.LocalSignParty
	parties        []p2p.Party
	sessionManager *p2p.SessionManager

	chainType   bridgeTypes.ChainType
	addSigOp    resharingTypes.ContractOperation
	removeSigOp resharingTypes.ContractOperation

	logger *logan.Entry
}

func NewUpdateContractHandler(
	self tss.LocalSignParty,
	parties []p2p.Party,
	sessionManager *p2p.SessionManager,
	chainType bridgeTypes.ChainType,
	addSignerOperation resharingTypes.ContractOperation,
	removeSignerOperation resharingTypes.ContractOperation,
	logger *logan.Entry,
) *UpdateContractHandler {
	return &UpdateContractHandler{
		self:           self,
		parties:        parties,
		sessionManager: sessionManager,
		chainType:      chainType,
		addSigOp:       addSignerOperation,
		removeSigOp:    removeSignerOperation,
		logger:         logger.WithField("component", fmt.Sprintf("contract_update_%s", chainType)),
	}
}

func (r *UpdateContractHandler) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	// no need to recover state
	return false, nil
}

func (r *UpdateContractHandler) MaxHandleDuration() time.Duration {
	// includes time to perform two signing operations and session init delays
	return 2*session.BoundarySign + 2*time.Second
}

func (r *UpdateContractHandler) Handle(ctx context.Context, state *resharingTypes.State) error {
	addSignerSession := signing.NewSession(
		r.self,
		signing.SessionParams{
			Params: session.Params{
				Threshold: r.self.Threshold,
			},
			SigningData: r.addSigOp.CalculateHash(),
		},
		r.parties,
		r.logger.WithField("operation", "add_signer"),
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
			SigningData: r.removeSigOp.CalculateHash(),
		},
		r.parties,
		r.logger.WithField("operation", "remove_signer"),
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

	state.AddSignature(bridgeTypes.EpochChainSignatures{
		ChainType: r.chainType,
		EpochId:   state.Epoch,
		AddedSignature: &bridgeTypes.EpochSignature{
			Mod:       bridgeTypes.EpochSignatureMod_ADD,
			EpochId:   state.Epoch,
			Signature: r.addSigOp.ConvertSignature(addSignerResult),
			Data: &bridgeTypes.EpochSignatureData{
				NewSigner: r.addSigOp.Signer(),
				StartTime: uint64(r.addSigOp.StartTime().Unix()),
				EndTime:   uint64(r.addSigOp.Deadline().Unix()),
				Nonce:     new(big.Int).SetUint64(r.addSigOp.Nonce()).String(),
			},
		},
		RemovedSignature: &bridgeTypes.EpochSignature{
			Mod:       bridgeTypes.EpochSignatureMod_REMOVE,
			EpochId:   state.Epoch,
			Signature: r.removeSigOp.ConvertSignature(removeSignerResult),
			Data: &bridgeTypes.EpochSignatureData{
				NewSigner: r.removeSigOp.Signer(),
				StartTime: uint64(r.removeSigOp.StartTime().Unix()),
				EndTime:   uint64(r.removeSigOp.Deadline().Unix()),
				Nonce:     new(big.Int).SetUint64(r.removeSigOp.Nonce()).String(),
			},
		},
	})

	return nil
}
