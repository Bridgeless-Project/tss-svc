package connector

import (
	"context"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errTxNotFound = status.Error(codes.NotFound, "transaction not found")

func (c *Connector) GetDepositInfo(identifier *types.DepositIdentifier) (*bridgetypes.Transaction, error) {
	req := bridgetypes.QueryTransactionByIdRequest{
		ChainId: identifier.ChainId,
		TxHash:  identifier.TxHash,
		TxNonce: uint64(identifier.TxNonce),
	}

	resp, err := c.querier.TransactionById(context.Background(), &req)
	if err != nil {
		if errors.Is(err, errTxNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get transaction info")
	}

	return &resp.Transaction, nil
}
