package requests

import (
	"context"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	database "github.com/hyle-team/tss-svc/internal/db"
	types "github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func SubmitTx(ctxt context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	var (
		chains = ctx.Chains(ctxt)
		db     = ctx.DB(ctxt)
		logger = ctx.Logger(ctxt)
	)

	if identifier == nil {
		logger.Error("empty identifier")
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}
	err := validation.Errors{
		"tx_hash":  validation.Validate(identifier.TxHash, validation.Required),
		"chain_id": validation.Validate(identifier.ChainId, validation.Required),
		"tx_nonce": validation.Validate(identifier.TxNonce, validation.Required),
	}.Filter()

	if err != nil {
		return nil, err
	}

	chain, ok := chains[identifier.ChainId]
	if !ok {
		return &emptypb.Empty{}, apiTypes.ErrInvalidChainId
	}
	err = db.Transaction(func() error {
		id := common.FormDepositIdentifier(identifier, chain.Type)
		exists, err := common.CheckIfDepositExists(id, db)
		if err != nil {
			return err
		}
		if exists {
			return apiTypes.ErrTxAlreadySubmitted
		}
		if err := common.GetDepositData(id, chain, db); err != nil {
			if errors.Is(err, database.ErrAlreadySubmitted) {
				logger.Debug("error inserting")
				return apiTypes.ErrTxAlreadySubmitted
			}

			logger.WithError(err).Error("failed to get deposit data")
			return apiTypes.ErrInternal
		}

		return nil
	})
	if err != nil {

		return nil, err
	}

	return nil, nil
}
