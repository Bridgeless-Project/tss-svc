package connector

import (
	"context"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
)

func (c *Connector) GetStopListTx(identifier *types.DepositIdentifier) (*bridgetypes.Transaction, error) {
	req := bridgetypes.QueryGetStopListTxById{
		ChainId: identifier.ChainId,
		TxHash:  identifier.TxHash,
		TxNonce: uint64(identifier.TxNonce),
	}

	resp, err := c.querier.GetStopListTxsById(context.Background(), &req)
	if err != nil {
		if errors.Is(err, errTxNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get stop list tx")
	}

	return &resp.Transaction, nil
}
