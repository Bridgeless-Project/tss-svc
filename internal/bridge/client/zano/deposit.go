package zano

import (
	"encoding/hex"
	"encoding/json"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"

	zanoTypes "github.com/hyle-team/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

type destinationData struct {
	Address string `json:"dst_add"`
	ChainId string `json:"dst_net_id"`
}

func (p *proxy) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	transaction, _, err := p.GetTransaction(id.TxHash, true, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction")
	}
	if transaction == nil {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	if err = p.validateConfirmations(transaction.Height); err != nil {
		return nil, errors.Wrap(err, "failed to validate confirmations")
	}

	if !transaction.Ado.IsValidAssetBurn() {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	if len(transaction.ServiceEntries) < id.TxNonce+1 {
		return nil, bridgeTypes.ErrDepositNotFound
	}
	addr, chainId, err := parseDestinationData(transaction.ServiceEntries[id.TxNonce])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse destination data")
	}

	var depositor string
	if len(transaction.RemoteAddresses) > 0 {
		depositor = transaction.RemoteAddresses[0]
	}

	return &db.DepositData{
		DepositIdentifier:  id,
		DestinationChainId: chainId,
		DestinationAddress: addr,
		SourceAddress:      depositor,
		DepositAmount:      transaction.Ado.OptAmount,
		TokenAddress:       *transaction.Ado.OptAssetId,
		Block:              int64(transaction.Height),
	}, nil
}

func (p *proxy) validateConfirmations(txHeight uint64) error {
	if txHeight == 0 {
		return bridgeTypes.ErrTxPending
	}

	currentHeight, err := p.chain.Client.CurrentHeight()
	if err != nil {
		return errors.Wrap(err, "failed to get current height")
	}

	if currentHeight-txHeight < p.chain.Confirmations {
		return bridgeTypes.ErrTxNotConfirmed
	}

	return nil
}

func parseDestinationData(entry zanoTypes.ServiceEntry) (addr, chainId string, err error) {
	raw, err := hex.DecodeString(entry.Body)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to decode hex body")
	}

	var dstData destinationData
	if err = json.Unmarshal(raw, &dstData); err != nil {
		return "", "", errors.Wrap(err, "failed to unmarshal json data")
	}

	return dstData.Address, dstData.ChainId, nil
}
