package zano

import (
	"encoding/hex"
	"encoding/json"

	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/db"

	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

func (p *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
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

	if int64(len(transaction.ServiceEntries)) < id.TxNonce+1 {
		return nil, bridgeTypes.ErrDepositNotFound
	}
	depositMemo, err := parseDepositMemo(transaction.ServiceEntries[id.TxNonce])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse destination data")
	}

	var depositor string
	if len(transaction.RemoteAddresses) > 0 {
		depositor = transaction.RemoteAddresses[0]
	}

	return &db.DepositData{
		DepositIdentifier:  id,
		DestinationChainId: depositMemo.ChainId,
		DestinationAddress: depositMemo.Address,
		ReferralId:         depositMemo.ReferralId,
		SourceAddress:      depositor,
		DepositAmount:      transaction.Ado.OptAmount,
		TokenAddress:       *transaction.Ado.OptAssetId,
		Block:              int64(transaction.Height),
	}, nil
}

func (p *Client) validateConfirmations(txHeight uint64) error {
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

func parseDepositMemo(entry zanoTypes.ServiceEntry) (*DepositMemo, error) {
	raw, err := hex.DecodeString(entry.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode hex body")
	}

	var depositMemo DepositMemo
	if err = json.Unmarshal(raw, &depositMemo); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal json data")
	}

	if err = depositMemo.Validate(); err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, err.Error())
	}
 
	return &depositMemo, nil
}
