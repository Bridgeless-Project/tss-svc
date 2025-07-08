package ton

import (
	"context"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/tlb"
)

func (c *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	tx, err := c.getTxByLtHash(uint64(id.TxNonce), id.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx")
	}

	data, err := c.parseDepositData(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse deposit data")
	}

	data.TxHash = id.TxHash
	data.DepositIdentifier = id

	return data, nil
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
	if tx.OutMsgCount == 0 {
		return nil, chain.ErrDepositNotFound
	}

	msgs, err := tx.IO.Out.ToSlice()
	if err != nil {
		return nil, errors.Wrap(err, "error getting IO out msgs")
	}

	var depositData *db.DepositData
	for _, msg := range msgs {
		if msg.MsgType != tlb.MsgTypeExternalOut {
			continue
		}

		opCode, err := parseMsgOpCode(msg.AsExternalOut().Body.BeginParse())
		if err != nil {
			return nil, errors.Wrap(err, "error parsing external out msg")
		}

		switch opCode {
		case depositNativeOpCode:
			content, err := c.parseDepositNativeBody(msg.AsExternalOut().Body)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing deposit native content from msg")
			}

			depositData = formNativeDepositData(content, tx)
			break

		case depositJettonOpCode:
			content, err := c.parseDepositJettonBody(msg.AsExternalOut().Body)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing deposit jetton content from msg")
			}

			depositData = formJettonDepositData(content, tx)
		default:
			return nil, errors.Wrap(chain.ErrUnsupportedEvent, "provided event is not supported deposit event")
		}

	}

	return depositData, nil
}
