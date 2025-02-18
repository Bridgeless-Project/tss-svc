package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/pkg/errors"
)

const notFoundErrorMessage = "not found"

func (p *Client) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, *common.Address, error) {
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
