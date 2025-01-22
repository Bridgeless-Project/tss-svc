package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"

	"github.com/pkg/errors"
)

const notFoundErrorMessage = "not found"

func (p *client) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, *common.Address, error) {
	ctx := context.Background()
	tx, pending, err := p.chain.Rpc.TransactionByHash(ctx, txHash)
	if err != nil {
		if err.Error() == notFoundErrorMessage {
			return nil, nil, bridgeTypes.ErrTxNotFound
		}

		return nil, nil, errors.Wrap(err, "failed to get transaction by hash")
	}
	if pending {
		return nil, nil, bridgeTypes.ErrTxPending
	}
	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get tx sender")
	}

	receipt, err := p.chain.Rpc.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get tx receipt")
	}
	if receipt == nil {
		return nil, nil, errors.New("receipt is nil")
	}

	return receipt, &from, nil
}

func (p *client) GetTransactionStatus(txHash string) (bridgeTypes.TransactionStatus, error) {
	receipt, _, err := p.GetTransactionReceipt(common.HexToHash(txHash))
	if err != nil {
		if errors.Is(err, bridgeTypes.ErrTxPending) {
			return bridgeTypes.TransactionStatusPending, nil
		}
		if errors.Is(err, bridgeTypes.ErrTxNotFound) {
			return bridgeTypes.TransactionStatusNotFound, nil
		}

		return bridgeTypes.TransactionStatusUnknown, errors.Wrap(err, "failed to get transaction receipt")
	}

	switch receipt.Status {
	case ethTypes.ReceiptStatusSuccessful:
		return bridgeTypes.TransactionStatusSuccessful, nil
	case ethTypes.ReceiptStatusFailed:
		return bridgeTypes.TransactionStatusFailed, nil
	default:
		return bridgeTypes.TransactionStatusUnknown, nil
	}
}
