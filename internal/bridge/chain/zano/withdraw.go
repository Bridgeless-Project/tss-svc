package zano

import (
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

func (p *Client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(bridge.ZeroAmount) != 1 {
		return false
	}

	return true
}

func (p *Client) EmitAssetUnsigned(data db.Deposit) (*zanoTypes.EmitAssetResponse, error) {
	amount, ok := new(big.Int).SetString(data.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to convert withdrawal amount")
	}

	destination := zanoTypes.Destination{
		Address: data.Receiver,
		Amount:  amount,
		// leaving empty here as this field overrides by function asset parameter
		AssetID: "",
	}

	return p.chain.Client.EmitAsset(data.WithdrawalToken, destination)
}

func (p *Client) TransferAssetOwnershipUnsigned(assetId, newOwnerPubKey string, isEthKey bool) (*zanoTypes.TransferAssetOwnershipResponse, error) {
	return p.chain.Client.TransferAssetOwnership(assetId, newOwnerPubKey, isEthKey)
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

func (p *Client) SendSignedTransaction(signedTx SignedTransaction) (string, error) {
	response, err := p.chain.Client.SendExtSignedAssetTX(
		signedTx.Signature,
		signedTx.ExpectedTxHash,
		signedTx.FinalizedTx,
		signedTx.Data,
		false,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to send signed transaction")
	}

	if response.Status != zanoTypes.StatusSendExtSignedAssetTxOk {
		return "", errors.Errorf("unexpected send signed transaction status: %s", response.Status)
	}

	return bridge.HexPrefix + signedTx.ExpectedTxHash, nil
}
