package connector

import (
	"context"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"
)

func (c *Connector) GetCommission(epoch uint32, tokenId uint64) (*bridgeTypes.Commission, error) {
	req := bridgeTypes.QueryGetCommissionByToken{
		EpochId: epoch,
		TokenId: tokenId,
	}

	resp, err := c.querier.GetCommissionByToken(context.Background(), &req)
	if err != nil {
		if errors.Is(err, bridgeTypes.ErrCommissionNotFound.GRPCStatus().Err()) {
			return nil, core.ErrCommissionNotFound
		}

		return nil, errors.Wrap(err, "failed to get commission info")
	}

	return &resp.Commission, nil
}
