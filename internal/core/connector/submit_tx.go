package core

import (
	bridgetypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"strings"
)

func (c *Connector) SubmitDeposits(depositTxs ...bridgetypes.Transaction) error {
	if len(depositTxs) == 0 {
		return nil
	}

	msg := bridgetypes.NewMsgSubmitTransactions(c.settings.Account.CosmosAddress().String(), depositTxs...)
	err := c.submitMsgs(msg)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), bridgetypes.ErrTranscationAlreadySubmitted.Error()) {
		return types.ErrTransactionAlreadySubmitted
	}

	return errors.Wrap(err, "failed to submit deposits")
}
