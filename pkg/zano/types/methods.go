package types

const (
	// wallet methods
	WalletMethodSearchForTransactions2 = "search_for_transactions2"
	WalletMethodEmitAsset              = "emit_asset"
	WalletMethodTransferAssetOwnership = "transfer_asset_ownership"
	WalletMethodBurnAsset              = "burn_asset"
	WalletMethodTransfer               = "transfer"
	WalletMethodSendExtSignedAssetTx   = "send_ext_signed_asset_tx"
	WalletMethodGetWalletInfo          = "get_wallet_info"

	StatusSendExtSignedAssetTxOk = "OK"

	// node methods
	NodeMethodDecryptTxDetails = "decrypt_tx_details"
	NodeMethodGetHeight        = "getheight"
	NodeMethodGetAssetInfo     = "get_asset_info"
)
