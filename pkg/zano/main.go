package zano

import (
	"github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

const (
	defaultMixin = 15
	defaultFee   = "10000000000"
)

type Sdk struct {
	client *Client
}

func NewSDK(walletRPC, nodeRPC string) *Sdk {
	return &Sdk{
		client: NewClient(walletRPC, nodeRPC),
	}
}

// Transfer Make new payment transaction from the wallet
// service []types.ServiceEntry can be empty.
// Wallet rpc api method
func (z Sdk) Transfer(comment string, service []types.ServiceEntry, destinations []types.Destination) (*types.TransferResponse, error) {
	if service == nil || len(service) == 0 {
		service = []types.ServiceEntry{}
	}
	if destinations == nil || len(destinations) == 0 {
		return nil, errors.New("destinations must be non-empty")
	}
	req := types.TransferParams{
		Comment:                 comment,
		Destinations:            destinations,
		ServiceEntries:          service,
		Fee:                     defaultFee,
		HideReceiver:            true,
		Mixin:                   defaultMixin,
		PaymentID:               "",
		PushPayer:               false,
		ServiceEntriesPermanent: true,
	}

	resp := new(types.TransferResponse)
	if err := z.client.Call(types.WalletMethodTransfer, resp, req, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// GetTransactions Search for transactions in the wallet by few parameters
// Pass a hash without 0x prefix
// If past empty string instead of a hash node will return all tx for this wallet
// wallet rpc api method
func (z Sdk) GetTransactions(txid string) (*types.GetTxResponse, error) {
	req := types.GetTxParams{
		FilterByHeight: false,
		In:             true,
		MaxHeight:      0,
		MinHeight:      0,
		Out:            true,
		Pool:           true,
		TxID:           txid,
	}
	resp := new(types.GetTxResponse)
	if err := z.client.Call(types.WalletMethodSearchForTransactions, resp, req, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// EmitAsset Emmit new coins of the asset, that is controlled by this wallet.
// assetId must be non-empty and without prefix 0x
// wallet rpc api method
func (z Sdk) EmitAsset(assetId string, destinations ...types.Destination) (*types.EmitAssetResponse, error) {
	if len(destinations) == 0 {
		return nil, errors.New("destinations must be non-empty")
	}

	req := types.EmitAssetParams{
		AssetID:                assetId,
		Destinations:           destinations,
		DoNotSplitDestinations: false,
	}

	resp := new(types.EmitAssetResponse)
	if err := z.client.Call(types.WalletMethodEmitAsset, resp, req, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// TransferAssetOwnership Transfer asset ownership to another wallet.
// assetId must be non-empty and without prefix 0x
// wallet rpc api method
func (z Sdk) TransferAssetOwnership(assetId, newOwnerPubKey string, isEthKey bool) (*types.TransferAssetOwnershipResponse, error) {
	req := types.TransferAssetOwnershipParams{
		AssetID: assetId,
	}
	if isEthKey {
		req.NewOwnerEthPubKey = &newOwnerPubKey
	} else {
		req.NewOwner = &newOwnerPubKey
	}

	resp := new(types.TransferAssetOwnershipResponse)
	if err := z.client.Call(types.WalletMethodTransferAssetOwnership, resp, req, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// BurnAsset Burn some owned amount of the coins for the given asset.
// https://docs.zano.org/docs/build/rpc-api/wallet-rpc-api/burn_asset/
// assetId must be non-empty and without prefix 0x
// wallet rpc api method
func (z Sdk) BurnAsset(assetId string, amount string) (*types.BurnAssetResponse, error) {
	req := types.BurnAssetParams{
		AssetID:    assetId,
		BurnAmount: amount,
	}

	resp := new(types.BurnAssetResponse)
	if err := z.client.Call(types.WalletMethodBurnAsset, resp, req, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// TxDetails Decrypts transaction private information. Should be used only with your own local daemon for security reasons.
// node rpc api method
func (z Sdk) TxDetails(outputAddress []string, txBlob, txID, txSecretKey string) (*types.DecryptTxDetailsResponse, error) {
	req := types.DecryptTxDetailsParams{
		OutputsAddresses: outputAddress,
		TxBlob:           txBlob,
		TxID:             txID,
		TxSecretKey:      txSecretKey,
	}

	resp := new(types.DecryptTxDetailsResponse)
	if err := z.client.Call(types.NodeMethodDecryptTxDetails, resp, req, false); err != nil {
		return nil, err
	}

	return resp, nil
}

// SendExtSignedAssetTX Inserts externally made asset ownership signature into the given transaction and broadcasts it.
// wallet rpc api method
func (z Sdk) SendExtSignedAssetTX(ethSig, expectedTXID, finalizedTx, unsignedTx string, unlockTransfersOnFail bool) (*types.SendExtSignedAssetTXResult, error) {
	req := types.SendExtSignedAssetTXParams{
		EthSig:                ethSig,
		ExpectedTxID:          expectedTXID,
		FinalizedTx:           finalizedTx,
		UnlockTransfersOnFail: unlockTransfersOnFail,
		UnsignedTx:            unsignedTx,
	}

	resp := new(types.SendExtSignedAssetTXResult)
	if err := z.client.Call(types.WalletMethodSendExtSignedAssetTx, resp, req, true); err != nil {
		return nil, err
	}

	return resp, nil
}

// CurrentHeight Get current blockchain height
// node rpc api method
func (z Sdk) CurrentHeight() (uint64, error) {
	resp := new(types.GetHeightResponse)
	err := z.client.CallRaw(types.NodeMethodGetHeight, resp)
	if err != nil {
		return 0, err
	}

	if resp.Status != "OK" {
		return 0, errors.Errorf("unexpected status: %s", resp.Status)
	}

	return resp.Height, err
}

func (z Sdk) GetWalletInfo() (*types.GetWalletInfoResponse, error) {
	resp := new(types.GetWalletInfoResponse)
	err := z.client.Call(types.WalletMethodGetWalletInfo, resp, nil, true)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (z Sdk) GetAssetInfo(assetId string) (*types.GetAssetInfoResponse, error) {
	req := types.GetAssetInfoRequest{AssetId: assetId}
	resp := new(types.GetAssetInfoResponse)

	if err := z.client.Call(types.NodeMethodGetAssetInfo, resp, req, false); err != nil {
	}
}
