package types

import (
	"encoding/base64"
	"encoding/json"
	"math/big"

	"github.com/pkg/errors"
)

type extraEntryType string

const (
	extraFieldAssetDescriptorBase                = "asset_descriptor_base"
	extraEntryAssetDescriptor     extraEntryType = "asset_descriptor"
)

type ExtraEntry struct {
	Type  extraEntryType
	Value interface{}
}

type TxExtra []ExtraEntry

// UnmarshalJSON implements the json.Unmarshaler interface for TxInExtra.
// Currently, it only handles the "asset_descriptor_base" extra entry.
// If you need to handle more entries, you can extend this function.
func (e *TxExtra) UnmarshalJSON(data []byte) error {
	var rawItems []map[string]interface{}
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return err
	}

	for _, item := range rawItems {
		value, exists := item[extraFieldAssetDescriptorBase]
		if !exists {
			continue
		}

		rawJson, err := json.Marshal(value)
		if err != nil {
			return errors.Wrap(err, "failed to marshal asset descriptor raw JSON")
		}

		var assetDescriptor AssetDescriptorBase
		if err = json.Unmarshal(rawJson, &assetDescriptor); err != nil {
			return errors.Wrap(err, "failed to unmarshal asset descriptor")
		}

		*e = append(*e, ExtraEntry{Type: extraEntryAssetDescriptor, Value: assetDescriptor})
	}

	return nil
}

type TxInJson struct {
	Aggregated struct {
		TxExtra TxExtra `json:"extra"`
		// Add other fields as needed
	} `json:"AGGREGATED"`
}
type AssetDescriptorBase struct {
	OperationType uint64           `json:"operation_type"`
	OptAssetId    *string          `json:"opt_asset_id"`
	OptDescriptor *AssetDescriptor `json:"opt_descriptor"`
	// Add other fields as needed
}

func (a *AssetDescriptorBase) IsValidTransferAssetOwnershipOperation() bool {
	return a.OperationType == OperationTypeTransferAssetOwnership && a.OptAssetId != nil && a.OptDescriptor != nil
}

type AssetDescriptor struct {
	Version        uint64   `json:"VERSION"`
	TotalMaxSupply *big.Int `json:"total_max_supply"`
	CurrentSupply  *big.Int `json:"current_supply"`
	DecimalPoint   int      `json:"decimal_point"`
	Ticker         string   `json:"ticker"`
	FullName       string   `json:"full_name"`
	MetaInfo       string   `json:"meta_info"`
	Owner          string   `json:"owner"`
	HiddenSupply   bool     `json:"hidden_supply"`
	OwnerEthPubKey string   `json:"owner_eth_pub_key"`
}

func (r *DecryptTxDetailsResponse) GetAssetInfo() (*AssetDescriptorBase, error) {
	rawJsonTx, err := base64.StdEncoding.DecodeString(r.TxInJSON)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 transaction")
	}

	var tx TxInJson
	if err = json.Unmarshal(rawJsonTx, &tx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}

	for _, extra := range tx.Aggregated.TxExtra {
		if extra.Type == extraEntryAssetDescriptor {
			assetDescriptor, ok := extra.Value.(AssetDescriptorBase)
			if !ok {
				return nil, errors.New("failed to cast asset descriptor")
			}
			return &assetDescriptor, nil
		}
	}

	return nil, errors.New("asset info not found")
}
