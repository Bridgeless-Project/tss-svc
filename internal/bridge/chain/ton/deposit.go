package ton

import (
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/xssnick/tonutils-go/tlb"
)

func (p *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	return nil, nil
}

func (c *Client) getTxByLtHash(lt uint64, txHash string) (*tlb.Transaction, error) {
	// TODO: Implement with/without sender address

	return nil, nil
}
