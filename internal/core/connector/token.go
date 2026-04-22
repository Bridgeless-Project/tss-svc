package connector

import (
	"context"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pkg/errors"
)

func (c *Connector) GetToken(id uint64) (*bridgeTypes.Token, error) {
	req := bridgeTypes.QueryGetTokenById{Id: id}

	resp, err := c.querier.GetTokenById(context.Background(), &req)
	if err != nil {
		if errors.Is(err, bridgeTypes.ErrTokenInfoNotFound.GRPCStatus().Err()) {
			return nil, core.ErrTokenInfoNotFound
		}

		return nil, errors.Wrap(err, "failed to get token info")
	}

	return &resp.Token, nil
}

func (c *Connector) GetTokens() ([]bridgeTypes.Token, error) {
	req := bridgeTypes.QueryGetTokens{
		Pagination: &query.PageRequest{
			Limit: query.MaxLimit,
		},
	}

	resp, err := c.querier.GetTokens(context.Background(), &req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tokens")
	}

	return resp.Tokens, nil
}
