package connector

import (
	"context"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
)

func (c *Connector) SetEpochPubKey(epoch uint32, pubKey string) error {
	msg := bridgeTypes.NewMsgSetEpochPubKey(c.account.CosmosAddress().String(), pubKey, epoch)

	err := c.submitMsgs(context.Background(), msg)
	if err == nil {
		return nil
	}
	// no new error is defined for this case in the core module
	if strings.Contains(err.Error(), bridgeTypes.ErrTranscationAlreadySubmitted.Error()) {
		return core.ErrTransactionAlreadySubmitted
	}

	return nil
}

func (c *Connector) GetEpochPubKey(epoch uint32) (string, error) {
	req := bridgeTypes.QueryGetEpochPubKey{EpochId: epoch}

	resp, err := c.querier.GetEpochPubKey(context.Background(), &req)
	if err != nil {
		if errors.Is(err, bridgeTypes.ErrEpochNotFound.GRPCStatus().Err()) {
			return "", core.ErrEpochNotFound
		}

		return "", errors.Wrap(err, "failed to get epoch pubkey")
	}

	return resp.PubKey, nil
}
