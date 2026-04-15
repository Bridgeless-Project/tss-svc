package connector

import (
	"context"
	"strings"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	swaptypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/swap/types"
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

func (c *Connector) SubmitSwaps(ctx context.Context, depositsSwapTxs *swaptypes.SwapTransaction, isBridgeTx bool) error {
	if depositsSwapTxs == nil {
		return nil
	}

	msg := swaptypes.NewMsgSubmitSwapTx(c.account.CosmosAddress().String(), depositsSwapTxs, isBridgeTx)
	err := c.submitMsgs(ctx, msg)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), swaptypes.ErrAlreadySubmitted.Error()) {
		return core.ErrTransactionAlreadySubmitted
	}

	return errors.Wrap(err, "failed to submit swap deposits")
}
