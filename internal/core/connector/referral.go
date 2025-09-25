package connector

import (
	"context"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/pkg/errors"
)

func (c *Connector) GetReferralById(id uint16) (*bridgetypes.Referral, error) {
	req := bridgetypes.QueryGetReferralById{
		ReferralId: uint32(id),
	}

	resp, err := c.querier.GetReferralById(context.Background(), &req)
	if err != nil {
		if errors.Is(err, bridgetypes.ErrReferralNotFound.GRPCStatus().Err()) {
			return nil, core.ErrReferralNotFound
		}

		return nil, errors.Wrap(err, "failed to get referral info")
	}

	return &resp.Referral, nil
}
