package connector

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	bridgetypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/pkg/errors"
)

func (c *Connector) GetTokenInfo(chainId string, addr string) (bridgetypes.TokenInfo, error) {
	req := bridgetypes.QueryGetTokenInfo{
		Chain:   chainId,
		Address: addr,
	}

	resp, err := c.querier.GetTokenInfo(context.Background(), &req)
	if err != nil {
		if errors.Is(err, bridgetypes.ErrTokenInfoNotFound.GRPCStatus().Err()) {
			return bridgetypes.TokenInfo{}, core.ErrSourceTokenInfoNotFound
		}

		return bridgetypes.TokenInfo{}, errors.Wrap(err, "failed to get token info")
	}

	return resp.Info, nil
}
