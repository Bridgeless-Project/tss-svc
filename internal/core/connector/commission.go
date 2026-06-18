package connector

import (
	"context"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pkg/errors"
)

func (c *Connector) GetCommission(epoch uint32, tokenId uint64, blockHeight ...int64) (*bridgeTypes.Commission, error) {
	req := bridgeTypes.QueryGetCommissionByToken{
		EpochId: epoch,
		TokenId: tokenId,
	}

	ctx := context.Background()
	if len(blockHeight) > 0 {
		ctx = historyCtx(ctx, blockHeight[0])
	}

	resp, err := c.querier.GetCommissionByToken(ctx, &req)
	if err != nil {
		if errors.Is(err, bridgeTypes.ErrCommissionNotFound.GRPCStatus().Err()) {
			return nil, core.ErrCommissionNotFound
		}

		return nil, errors.Wrap(err, "failed to get commission info")
	}

	return &resp.Commission, nil
}

func (c *Connector) GetCommissions(epoch uint32, blockHeight ...int64) ([]bridgeTypes.Commission, error) {
	req := bridgeTypes.QueryGetCommissions{
		EpochId:    epoch,
		Pagination: &query.PageRequest{Limit: query.MaxLimit},
	}

	ctx := context.Background()
	if len(blockHeight) > 0 {
		ctx = historyCtx(ctx, blockHeight[0])
	}

	resp, err := c.querier.GetCommissions(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get commissions")
	}

	return resp.Commissions, nil
}

func MapCommissions(commissions []bridgeTypes.Commission) map[uint64]bridgeTypes.Commission {
	commissionMap := make(map[uint64]bridgeTypes.Commission, len(commissions))
	for _, commission := range commissions {
		commissionMap[commission.TokenId] = commission
	}

	return commissionMap
}
