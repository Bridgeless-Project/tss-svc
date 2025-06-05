package ton

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/tlb"
)

func (c *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	tx, err := c.getTxByLtHash(uint64(id.TxNonce), id.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx")
	}

	return &db.DepositData{}, nil

}

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

func (c *Client) parseDepositData(tx *tlb.Transaction) (*db.DepositData, error) {
	var (
		internalData *db.DepositData
		externalData *db.DepositData
	)

	if tx.OutMsgCount == 0 {
		return nil, chain.ErrDepositNotFound
	}

	msgs, err := tx.IO.Out.ToSlice()
	if err != nil {
		return nil, errors.Wrap(err, "error getting IO out msgs")
	}

	for _, msg := range msgs {
		if msg.MsgType != tlb.MsgTypeExternalOut {
			// if out msg is not ExternalOut log - skip it
			continue
		}

	}

	return &data, nil
}

func parseExternalOutData(msg *tlb.ExternalMessageOut) (*db.DepositData, error) {
	opBig, err := msg.Payload().BeginParse().PreloadBigUInt(32)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing op code")
	}

	if hexutil.Encode(opBig.Bytes()) == DepositedCode {
		return &db.DepositData{}, nil
	}

	var content

	return nil, chain.ErrDepositNotFound
}

func parseInternalData(msg *tlb.InternalMessage) (*db.DepositData, error) {

}
