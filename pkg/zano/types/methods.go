package types

const (
	// wallet methods
	SearchForTransactionsMethod  = "search_for_transactions"
	EmitAssetMethod              = "emit_asset"
	TransferAssetOwnershipMethod = "transfer_asset_ownership"
	BurnAssetMethod              = "burn_asset"
	TransferMethod               = "transfer"
	SendExtSignedAssetTxMethod   = "send_ext_signed_asset_tx"

	// node methods
	DecryptTxDetailsMethod = "decrypt_tx_details"
	GetHeightMethod        = "getheight"
)
