package connector

import (
	"context"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
)

func (c *Connector) GetBridgeParams() (*bridgeTypes.Params, error) {
	response, err := c.querier.Params(context.Background(), &bridgeTypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return &response.Params, nil
}
