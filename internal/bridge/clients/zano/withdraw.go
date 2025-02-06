package zano

import (
	"math/big"

	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	zanoTypes "github.com/hyle-team/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

func (p *Client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(bridge.ZeroAmount) != 1 {
		return false
	}

	return true
}

func (p *Client) EmitAssetUnsigned(data db.Deposit) (*zanoTypes.EmitAssetResponse, error) {
	amount, ok := new(big.Int).SetString(*data.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to convert withdrawal amount")
	}

	destination := zanoTypes.Destination{
		Address: *data.Receiver,
		Amount:  amount,
		// leaving empty here as this field overrides by function asset parameter
		AssetID: "",
	}

	return p.chain.Client.EmitAsset(*data.WithdrawalToken, destination)
}

func (p *Client) DecryptTxDetails(data zanoTypes.DataForExternalSigning) (*zanoTypes.DecryptTxDetailsResponse, error) {
	return p.chain.Client.TxDetails(
		data.OutputsAddresses,
		data.UnsignedTx,
		// leaving empty as only unsignedTx OR txId should be specified, otherwise error
		"",
		data.TxSecretKey,
	)
}

func (p *Client) EmitAssetSigned(signedTx SignedTransaction) (string, error) {
	_, err := p.chain.Client.SendExtSignedAssetTX(
		signedTx.Signature,
		signedTx.ExpectedTxHash,
		signedTx.FinalizedTx,
		signedTx.Data,
		// TODO: investigate
		true,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to emit signed asset")
	}

	return bridge.HexPrefix + signedTx.ExpectedTxHash, nil
}
