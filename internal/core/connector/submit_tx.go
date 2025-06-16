package connector

import (
	"context"
	"strings"

	bridgetypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"
)

func (c *Connector) SubmitDeposits(ctx context.Context, depositTxs ...bridgetypes.Transaction) error {
	if len(depositTxs) == 0 {
		return nil
	}

	msg := bridgetypes.NewMsgSubmitTransactions(c.account.CosmosAddress().String(), depositTxs...)
	err := c.submitMsgs(ctx, msg)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), bridgetypes.ErrTranscationAlreadySubmitted.Error()) {
		return core.ErrTransactionAlreadySubmitted
	}

	return errors.Wrap(err, "failed to submit deposits")
}
