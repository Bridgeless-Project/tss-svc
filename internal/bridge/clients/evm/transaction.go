package evm

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/clients"
	"github.com/pkg/errors"
)

func (p *Client) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, *common.Address, error) {
	ctx := context.Background()

	// TODO: Change after Pectra upgrade
	tx, pending, from, err := p.getTxByHash(ctx, txHash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, nil, bridgeTypes.ErrTxNotFound
		}

		return nil, nil, errors.Wrap(err, "failed to get transaction by hash")
	}
	if pending {
		return nil, nil, bridgeTypes.ErrTxPending
	}

	if from == nil {
		// FIXME: Add support for EIP-1559 when it's available in the Ethereum client
		sender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get transaction sender")
		}
		from = &sender
	}

	receipt, err := p.chain.Rpc.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get tx receipt")
	}
	if receipt == nil {
		return nil, nil, errors.New("receipt is nil")
	}

	return receipt, from, nil
}

type rpcTx struct {
	tx *types.Transaction
	txExtraInfo
}

func (tx *rpcTx) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

// getTxByHash is a replacement for ethclient.Client.TransactionByHash that returns the transaction's sender address.
func (p *Client) getTxByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, from *common.Address, err error) {
	var raw *rpcTx
	err = p.chain.Rpc.Client().CallContext(ctx, &raw, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, false, nil, err
	} else if raw == nil {
		return nil, false, nil, ethereum.NotFound
	} else if _, r, _ := raw.tx.RawSignatureValues(); r == nil {
		return nil, false, nil, errors.New("server returned transaction without signature")
	}

	return raw.tx, raw.BlockNumber == nil, raw.From, nil
}
