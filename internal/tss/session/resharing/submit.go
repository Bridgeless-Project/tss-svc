package resharing

import (
	"context"
	"math/big"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ resharingTypes.Handler = &SubmitHandler{}

type SubmitHandler struct {
	connector *connector.Connector
	logger    *logan.Entry
}

func NewSubmitHandler(connector *connector.Connector, logger *logan.Entry) *SubmitHandler {
	return &SubmitHandler{
		connector: connector,
		logger:    logger,
	}
}

func (h *SubmitHandler) MaxHandleDuration() time.Duration {
	return time.Minute * 2
}

func (h *SubmitHandler) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	// TODO: check if was already submitted to core
	return false, nil
}

func (h *SubmitHandler) Handle(ctx context.Context, state *resharingTypes.State) error {
	signatures, addresses := h.stateToEpochData(state)

	cooldown := 0 * time.Second
	for {
		select {
		case <-ctx.Done():
			return errors.New("context cancelled while waiting for pubkey confirmation")
		case <-time.After(cooldown):
			cooldown = 5 * time.Second

			err := h.connector.SubmitEpochSignatures(signatures, addresses)
			if err == nil || errors.Is(err, core.ErrTransactionAlreadySubmitted) {
				return nil
			}

			h.logger.WithError(err).Error("failed to submit epoch data, will retry...")
		}
	}
}

func (h *SubmitHandler) stateToEpochData(state *resharingTypes.State) ([]bridgeTypes.EpochChainSignatures, []bridgeTypes.EpochBridgeAddress) {
	signatures := make([]bridgeTypes.EpochChainSignatures, 0)
	data := state.EvmData
	if data != nil {
		signatures = append(signatures, bridgeTypes.EpochChainSignatures{
			EpochId:   state.Epoch,
			ChainType: bridgeTypes.ChainType_EVM,
			AddedSignature: &bridgeTypes.EpochSignature{
				Mod:       bridgeTypes.EpochSignatureMod_ADD,
				EpochId:   state.Epoch,
				Signature: data.AddNewSignerSignature.Signature,
				Data: &bridgeTypes.EpochSignatureData{
					NewSigner: data.AddNewSignerSignature.Signer,
					StartTime: uint64(data.AddNewSignerSignature.StartTime),
					EndTime:   uint64(data.AddNewSignerSignature.Deadline),
					Nonce:     new(big.Int).SetUint64(data.AddNewSignerSignature.Nonce).String(),
				},
			},
			RemovedSignature: &bridgeTypes.EpochSignature{
				Mod:       bridgeTypes.EpochSignatureMod_REMOVE,
				EpochId:   state.Epoch,
				Signature: data.RemoveOldSignerSignature.Signature,
				Data: &bridgeTypes.EpochSignatureData{
					NewSigner: data.RemoveOldSignerSignature.Signer,
					StartTime: uint64(data.RemoveOldSignerSignature.StartTime),
					EndTime:   uint64(data.RemoveOldSignerSignature.Deadline),
					Nonce:     new(big.Int).SetUint64(data.RemoveOldSignerSignature.Nonce).String(),
				},
			},
		})
	}

	addresses := make([]bridgeTypes.EpochBridgeAddress, len(state.NewBridgeAddresses))
	for chainId, addr := range state.NewBridgeAddresses {
		addresses = append(addresses, bridgeTypes.EpochBridgeAddress{
			EpochId: state.Epoch,
			ChainId: chainId,
			Address: addr,
		})
	}

	return signatures, addresses
}
