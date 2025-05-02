package connector

import (
	"context"
	bridgeTypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/pkg/errors"
)

func (c *Connector) GetToken(id uint64) (bridgeTypes.Token, error) {
	req := bridgeTypes.QueryGetTokenById{Id: id}

	resp, err := c.querier.GetTokenById(context.Background(), &req)
	if err != nil {
		if errors.Is(err, bridgeTypes.ErrTokenInfoNotFound.GRPCStatus().Err()) {
			return bridgeTypes.Token{}, core.ErrTokenInfoNotFound
		}

		return bridgeTypes.Token{}, errors.Wrap(err, "failed to get token info")
	}

	return resp.Token, nil
}
