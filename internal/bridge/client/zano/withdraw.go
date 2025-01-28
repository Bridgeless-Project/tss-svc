package zano

import (
	"math/big"

	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	zanoTypes "github.com/hyle-team/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

func (p *client) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(bridge.ZeroAmount) != 1 {
		return false
	}

	return true
}

func (p *client) EmitAssetUnsigned(data db.Deposit) (*UnsignedTransaction, error) {
	amount, ok := new(big.Int).SetString(*data.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to convert withdrawal amount")
	}

	destination := zanoTypes.Destination{
		Address: *data.Receiver,
		Amount:  amount.Uint64(),
		// leaving empty here as this field overrides by function asset parameter
		AssetID: "",
	}

	raw, err := p.chain.Client.EmitAsset(*data.WithdrawalToken, destination)
	if err != nil {
		return nil, errors.Wrap(err, "failed to emit unsigned asset")
	}

	signingData := raw.DataForExternalSigning
	txDetails, err := p.chain.Client.TxDetails(
		signingData.OutputsAddresses,
		signingData.UnsignedTx,
		// leaving empty as only unsignedTx OR txId should be specified, otherwise error
		"",
		signingData.TxSecretKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tx details")
	}

	return &UnsignedTransaction{
		ExpectedTxHash: txDetails.VerifiedTxID,
		FinalizedTx:    signingData.FinalizedTx,
		Data:           signingData.UnsignedTx,
	}, nil
}

func (p *client) EmitAssetSigned(signedTx SignedTransaction) (string, error) {
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
