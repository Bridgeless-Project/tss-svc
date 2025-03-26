package grpc

import (
	"context"

	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	"github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss/session/acceptor"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (Implementation) SubmitWithdrawal(ctxt context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	var (
		clientsRepo   = ctx.Clients(ctxt)
		data          = ctx.DB(ctxt)
		logger        = ctx.Logger(ctxt)
		processor     = ctx.Fetcher(ctxt)
		coreConnector = ctx.CoreConnector(ctxt)
	)

	client, err := clientsRepo.Client(identifier.ChainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "unsupported chain")
	}
	if err = common.ValidateIdentifier(identifier, client); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	existingDeposit, err := coreConnector.GetDepositInfo(identifier)
	if err != nil {
		logger.WithError(err).Error("error checking deposit info on core")
		return nil, ErrInternal
	}
	if existingDeposit != nil {
		return nil, status.Error(codes.AlreadyExists, "deposit already exists")
	}

	id := db.DepositIdentifier{
		ChainId: identifier.ChainId,
		TxHash:  identifier.TxHash,
		TxNonce: int(identifier.TxNonce),
	}
	deposit, err := data.Get(id)
	if err != nil {
		logger.WithError(err).Error("error getting deposit")
		return nil, ErrInternal
	}
	if deposit != nil {
		return nil, status.Error(codes.AlreadyExists, "deposit already exists")
	}

	deposit, err = processor.FetchDeposit(id)
	if err != nil {
		if clients.IsPendingDepositError(err) {
			return nil, ErrDepositPending
		}
		if clients.IsInvalidDepositError(err) || core.IsInvalidDepositError(err) {
			deposit = &db.Deposit{
				DepositIdentifier: id,
				WithdrawalStatus:  types.WithdrawalStatus_WITHDRAWAL_STATUS_INVALID,
			}
			if _, err = data.Insert(*deposit); err != nil {
				logger.WithError(err).Error("failed to process deposit")
				return nil, ErrInternal
			}

			return nil, status.Error(codes.InvalidArgument, "invalid deposit")
		}

		logger.WithError(err).Error("failed to fetch deposit")
		return nil, ErrInternal
	}

	if _, err = data.Insert(*deposit); err != nil {
		if errors.Is(err, db.ErrAlreadySubmitted) {
			return nil, ErrTxAlreadySubmitted
		}

		logger.WithError(err).Error("failed to save deposit")
		return nil, ErrInternal
	}

	raw, _ := anypb.New(&p2p.DepositDistributionData{DepositId: identifier})
	// broadcasting in a separate goroutine to avoid request blocking
	go ctx.Broadcaster(ctxt).Broadcast(&p2p.SubmitRequest{
		Sender:    ctx.Self(ctxt).String(),
		Type:      p2p.RequestType_RT_DEPOSIT_DISTRIBUTION,
		SessionId: acceptor.DepositAcceptorSessionIdentifier,
		Data:      raw,
	})

	return nil, nil
}
