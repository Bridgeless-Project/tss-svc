package resharing

import (
	"context"
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
	return false, nil
}

func (h *SubmitHandler) Handle(ctx context.Context, state *resharingTypes.State) error {
	signatures, addresses := h.stateToEpochData(state)

	retryCooldown := 0 * time.Second
	for {
		select {
		case <-ctx.Done():
			return errors.New("context cancelled while waiting for pubkey confirmation")
		case <-time.After(retryCooldown):
			retryCooldown = 5 * time.Second

			err := h.connector.SubmitEpochSignatures(state.Epoch, signatures, addresses)
			if err == nil || errors.Is(err, core.ErrTransactionAlreadySubmitted) {
				return nil
			}

			h.logger.WithError(err).Error("failed to submit epoch data, will retry...")
		}
	}
}

func (h *SubmitHandler) stateToEpochData(state *resharingTypes.State) ([]bridgeTypes.EpochChainSignatures, []bridgeTypes.EpochBridgeAddress) {
	addresses := make([]bridgeTypes.EpochBridgeAddress, 0, len(state.NewBridgeAddresses))
	for chainId, addr := range state.NewBridgeAddresses {
		addresses = append(addresses, bridgeTypes.EpochBridgeAddress{
			EpochId: state.Epoch,
			ChainId: chainId,
			Address: addr,
		})
	}

	return state.Signatures, addresses
}
