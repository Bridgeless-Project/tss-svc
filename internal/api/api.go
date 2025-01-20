package api

import (
	"context"
	validation "github.com/go-ozzo/ozzo-validation"
	ctxt "github.com/hyle-team/tss-svc/internal/api/ctx"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	db2 "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"strconv"
)

var _ APIServer = &server{}

type server struct {
}

func (s *server) SubmitWithdrawal(ctx context.Context, identifier *types.DepositIdentifier) (*emptypb.Empty, error) {
	var (
		db     = ctxt.DB(ctx)
		logger = ctxt.Logger(ctx)
	)
	if identifier == nil {
		logger.Error("empty identifier")
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}
	err := validation.Errors{
		"tx_hash":        validation.Validate(identifier.TxHash, validation.Required),
		"tx_event_index": validation.Validate(identifier.TxEventId, validation.Min(0)),
		"chain_id":       validation.Validate(identifier.ChainId, validation.Required),
		"tx_nonce":       validation.Validate(identifier.TxNonce, validation.Required),
	}.Filter()

	if err != nil {
		return &emptypb.Empty{}, err
	}

	err = db.Transaction(func() error {
		deposit := &db2.Deposit{
			DepositIdentifier: formDepositIdentifier(identifier),
			Status:            types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING,
		}

		if deposit.Id, err = db.Insert(*deposit); err != nil {
			if errors.Is(err, db2.ErrAlreadySubmitted) {
				return apiTypes.ErrTxAlreadySubmitted
			}

			logger.WithError(err).Error("failed to insert transaction")
			return apiTypes.ErrInternal
		}
		return nil
	})
	if err != nil {
		return &emptypb.Empty{}, err
	}
	return &emptypb.Empty{}, nil
}

func (s *server) CheckWithdrawal(ctx context.Context, identifier *types.DepositIdentifier) (*CheckWithdrawalResponse, error) {
	var (
		db     = ctxt.DB(ctx)
		logger = ctxt.Logger(ctx)
	)
	id := formDepositIdentifier(identifier)

	tx, err := db.Get(id)
	if err != nil {
		logger.WithError(err).Error("failed to get deposit")
		return nil, apiTypes.ErrInternal
	}
	if tx == nil {
		return nil, status.Error(codes.NotFound, "deposit not found")
	}

	return toStatusResponse(*tx), nil
}

func toStatusResponse(d db2.Deposit) *CheckWithdrawalResponse {
	result := &CheckWithdrawalResponse{
		DepositIdentifier: &types.DepositIdentifier{
			TxHash:    d.TxHash,
			TxEventId: int64(d.TxEventId),
			ChainId:   d.ChainId,
		},
		TransferData: &types.TransferData{
			Sender:           d.Depositor,
			Receiver:         *d.Receiver,
			DepositAmount:    *d.DepositAmount,
			WithdrawalAmount: *d.WithdrawalAmount,
			DepositAsset:     *d.DepositToken,
			WithdrawalAsset:  *d.WithdrawalToken,
			IsWrappedAsset:   strconv.FormatBool(*d.IsWrappedToken),
			DepositBlock:     *d.DepositBlock,
			Signature:        d.Signature,
		},
		WithdrawalStatus: d.Status,
		WithdrawalIdentifier: &types.WithdrawalIdentifier{
			TxHash:  *d.WithdrawalTxHash,
			ChainId: *d.WithdrawalChainId,
		},
	}
	return result
}

func formDepositIdentifier(identifier *types.DepositIdentifier) db2.DepositIdentifier {
	return db2.DepositIdentifier{
		TxHash:    identifier.TxHash,
		TxEventId: int(identifier.TxEventId),
		ChainId:   identifier.ChainId,
	}
}
