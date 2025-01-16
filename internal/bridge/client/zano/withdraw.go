package zano

import (
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	zanoTypes "github.com/hyle-team/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
	"math/big"
)

func (p *proxy) WithdrawalAmountValid(amount *big.Int) bool {
	if amount.Cmp(bridge.ZeroAmount) != 1 {
		return false
	}

	return true
}

func (p *proxy) EmitAssetUnsigned(data db.DepositData) (*UnsignedTransaction, error) {
	destination := zanoTypes.Destination{
		Address: data.DestinationAddress,
		Amount:  data.WithdrawalAmount.Uint64(),
		// leaving empty here as this field overrides by function asset parameter
		AssetID: "",
	}

	raw, err := p.chain.Client.EmitAsset(data.DestinationTokenAddress, destination)
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

func (p *proxy) EmitAssetSigned(signedTx SignedTransaction) (string, error) {
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
