package ton

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/tlb"
)

func (c *Client) getTxByLtHash(lt uint64, txHash string) (*tlb.Transaction, error) {
	txs, err := c.Client.ListTransactions(context.Background(), c.BridgeContractAddress, 1, lt, hexutil.MustDecode(txHash))
	if err != nil {
		return nil, errors.Wrap(err, "error getting deposit transaction")
	}
	if len(txs) == 0 {
		return nil, chain.ErrDepositNotFound
	}

	return txs[0], nil
}
