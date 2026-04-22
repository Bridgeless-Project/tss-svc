package types

import "math/big"

const (
	OperationTypeTransferAssetOwnership = 3
	OperationTypeAssetBurn              = 4
)

// wallet
type ServiceEntry struct {
	Body        string `json:"body"`
	Flags       int    `json:"flags"`
	Instruction string `json:"instruction"`
	Security    string `json:"security"`
	ServiceID   string `json:"service_id"`
}

type Destination struct {
	Address string   `json:"address"`
	Amount  *big.Int `json:"amount"`
	AssetID string   `json:"asset_id"`
}

type TransferParams struct {
	Comment                 string         `json:"comment"`
	Destinations            []Destination  `json:"destinations"`
	Fee                     string         `json:"fee"`
	HideReceiver            bool           `json:"hide_receiver"`
	Mixin                   int            `json:"mixin"`
	PaymentID               string         `json:"payment_id"`
	PushPayer               bool           `json:"push_payer"`
	ServiceEntries          []ServiceEntry `json:"service_entries"`
	ServiceEntriesPermanent bool           `json:"service_entries_permanent"`
}

type TransferResponse struct {
	TxHash        string `json:"tx_hash"`
	TxSize        int    `json:"tx_size"`
	TxUnsignedHex string `json:"tx_unsigned_hex"`
}

type GetTxParams struct {
	FilterByHeight bool   `json:"filter_by_height"`
	In             bool   `json:"in"`
	MaxHeight      int    `json:"max_height"`
	MinHeight      int    `json:"min_height"`
	Out            bool   `json:"out"`
	Pool           bool   `json:"pool"`
	TxID           string `json:"tx_id"`
}

type GetTxResponse struct {
	In   []Transaction `json:"in"`
	Out  []Transaction `json:"out"`
	Pool []Transaction `json:"pool"`
}

type Transaction struct {
	Ado                   *AssetDescriptorOperation `json:"ado,omitempty"`
	Amount                uint64                    `json:"amount"`
	Comment               string                    `json:"comment"`
	Contract              []Contract                `json:"contract"`
	EmployedEntries       interface{}               `json:"employed_entries"`
	Fee                   uint64                    `json:"fee"`
	Height                uint64                    `json:"height"`
	IsIncome              bool                      `json:"is_income"`
	IsMining              bool                      `json:"is_mining"`
	IsMixing              bool                      `json:"is_mixing"`
	IsService             bool                      `json:"is_service"`
	PaymentID             string                    `json:"payment_id"`
	RemoteAddresses       []string                  `json:"remote_addresses"`
	RemoteAliases         []string                  `json:"remote_aliases"`
	ServiceEntries        []ServiceEntry            `json:"service_entries"`
	ShowSender            bool                      `json:"show_sender"`
	Subtransfers          []SubTransfer             `json:"subtransfers"`
	Timestamp             int                       `json:"timestamp"`
	TransferInternalIndex int                       `json:"transfer_internal_index"`
	TxBlobSize            int                       `json:"tx_blob_size"`
	TxHash                string                    `json:"tx_hash"`
	TxType                int                       `json:"tx_type"`
	UnlockTime            int                       `json:"unlock_time"`
}

type Contract struct {
	CancelExpirationTime int             `json:"cancel_expiration_time"`
	ContractID           string          `json:"contract_id"`
	ExpirationTime       int             `json:"expiration_time"`
	Height               int             `json:"height"`
	IsA                  bool            `json:"is_a"`
	PaymentID            string          `json:"payment_id"`
	PrivateDetailes      PrivateDetailes `json:"private_detailes"`
	State                int             `json:"state"`
	Timestamp            int             `json:"timestamp"`
}

type PrivateDetailes struct {
	AAddr   string `json:"a_addr"`
	APledge int    `json:"a_pledge"`
	BAddr   string `json:"b_addr"`
	BPledge int    `json:"b_pledge"`
	C       string `json:"c"`
	T       string `json:"t"`
	ToPay   string `json:"to_pay"`
}

type SubTransfer struct {
	Amount   uint64 `json:"amount"`
	AssetID  string `json:"asset_id"`
	IsIncome bool   `json:"is_income"`
}

type EmitAssetParams struct {
	AssetID                string        `json:"asset_id"`
	Destinations           []Destination `json:"destinations"`
	DoNotSplitDestinations bool          `json:"do_not_split_destinations"`
}

type EmitAssetResponse struct {
	DataForExternalSigning DataForExternalSigning `json:"data_for_external_signing"`
	TxID                   string                 `json:"tx_id"`
}

type TransferAssetOwnershipParams struct {
	AssetID           string  `json:"asset_id"`
	NewOwnerEthPubKey *string `json:"new_owner_eth_pub_key,omitempty"`
	NewOwner          *string `json:"new_owner,omitempty"`
}

type TransferAssetOwnershipResponse struct {
	DataForExternalSigning DataForExternalSigning `json:"data_for_external_signing"`
	TxID                   string                 `json:"tx_id"`
}

type BurnAssetParams struct {
	AssetID    string `json:"asset_id"`
	BurnAmount string `json:"burn_amount"`
}

type BurnAssetResponse struct {
	DataForExternalSigning DataForExternalSigning `json:"data_for_external_signing"`
	TxID                   string                 `json:"tx_id"`
}

type DataForExternalSigning struct {
	FinalizedTx      string   `json:"finalized_tx"`
	OutputsAddresses []string `json:"outputs_addresses"`
	TxSecretKey      string   `json:"tx_secret_key"`
	UnsignedTx       string   `json:"unsigned_tx"`
}

type DecryptTxDetailsParams struct {
	OutputsAddresses []string `json:"outputs_addresses"`
	TxBlob           string   `json:"tx_blob"`
	TxID             string   `json:"tx_id"`
	TxSecretKey      string   `json:"tx_secret_key"`
}

type SendExtSignedAssetTXParams struct {
	EthSig                string `json:"eth_sig"`
	ExpectedTxID          string `json:"expected_tx_id"`
	FinalizedTx           string `json:"finalized_tx"`
	UnlockTransfersOnFail bool   `json:"unlock_transfers_on_fail"`
	UnsignedTx            string `json:"unsigned_tx"`
}

type SendExtSignedAssetTXResult struct {
	TransfersWereUnlocked bool   `json:"transfer_were_unlocked"`
	Status                string `json:"status"`
}

// RPC
type DecryptTxDetailsResponse struct {
	DecodedOutputs []TxOutput `json:"decoded_outputs"`
	Status         string     `json:"status"`
	TxInJSON       string     `json:"tx_in_json"`
	VerifiedTxID   string     `json:"verified_tx_id"`
}

type TxOutput struct {
	Address  string   `json:"address"`
	Amount   *big.Int `json:"amount"`
	AssetID  string   `json:"asset_id"`
	OutIndex int      `json:"out_index"`
}

type AssetDescriptorOperation struct {
	OperationType       uint64   `json:"operation_type"`
	OptAmount           *big.Int `json:"opt_amount"`
	OptAmountCommitment *string  `json:"opt_amount_commitment"`
	OptAssetId          *string  `json:"opt_asset_id"`
	Version             uint64   `json:"VERSION"`
}

func (o *AssetDescriptorOperation) IsValidAssetBurn() bool {
	if o == nil {
		return false
	}

	return o.IsAssetBurnOperation() && o.OptAmount != nil && o.OptAssetId != nil
}

func (o *AssetDescriptorOperation) IsAssetBurnOperation() bool {
	return o.OperationType == OperationTypeAssetBurn
}

type GetHeightResponse struct {
	Height uint64 `json:"height"`
	Status string `json:"status"`
}

type GetWalletInfoResponse struct {
	Address string `json:"address"`
	// TODO: add more fields if needed
}

type GetAssetInfoRequest struct {
	AssetId string `json:"asset_id"`
}
type GetAssetInfoResponse struct {
	AssetDescriptor AssetDescriptor `json:"asset_descriptor"`
}
