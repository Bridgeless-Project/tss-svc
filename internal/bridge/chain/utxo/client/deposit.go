package client

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/pkg/encoding"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/pkg/errors"
)

const (
	defaultDepositorAddressOutputIdx = 0
)

func (c *client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	tx, err := c.GetTransaction(id.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction")
	}
	if tx.BlockHash == "" {
		return nil, bridgeTypes.ErrTxPending
	}
	if tx.Confirmations < c.chain.Confirmations {
		return nil, bridgeTypes.ErrTxNotConfirmed
	}

	block, err := c.chain.Rpc.Node.GetBlockVerbose(tx.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block")
	}

	depositData, err := c.depositDecoder.Decode(tx, id.TxNonce)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode deposit data")
	}

	depositor, err := c.parseSenderAddress(tx.Vin[defaultDepositorAddressOutputIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get depositor")
	}

	return &db.DepositData{
		DepositIdentifier:  id,
		DestinationChainId: depositData.ChainId,
		DestinationAddress: depositData.Address,
		SourceAddress:      depositor,
		DepositAmount:      depositData.Amount,
		ReferralId:         depositData.ReferralId,
		// as Bitcoin does not have any other currencies
		TokenAddress: bridge.DefaultNativeTokenAddress,
		Block:        block.Height,
	}, nil
}

func (c *client) parseSenderAddress(in btcjson.Vin) (addr string, err error) {
	prevTx, err := c.GetTransaction(in.Txid)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get previous transaction %s", in.Txid)
	}
	if int(in.Vout) >= len(prevTx.Vout) {
		return "", errors.New("sender vout not found")
	}

	scriptRaw, err := hex.DecodeString(prevTx.Vout[in.Vout].ScriptPubKey.Hex)
	if err != nil {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	addrs, err := c.helper.ExtractScriptAddresses(scriptRaw)
	if err != nil {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	return addrs[0], nil
}

type DepositDecoder struct {
	helper          helper.UtxoHelper
	bridgeAddresses []string
}

type DepositData struct {
	Amount *big.Int
	DepositMemo
}

type DepositMemo struct {
	Address    string
	ChainId    string
	ReferralId uint16
}

const (
	PaddingByte      byte = 0x00
	chainIdLength         = 6
	referralIdLength      = 2
)

func NewDepositDecoder(helper helper.UtxoHelper, bridgeAddresses []string) *DepositDecoder {
	return &DepositDecoder{
		helper:          helper,
		bridgeAddresses: bridgeAddresses,
	}
}

func (d *DepositDecoder) Decode(tx *btcjson.TxRawResult, depositIdx int64) (*DepositData, error) {
	if depositIdx < 0 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionData, "invalid deposit index")
	}
	var (
		depositOutputIdx     = int(depositIdx)
		destinationOutputIdx = depositOutputIdx + 1
	)

	if depositOutputIdx < 0 || destinationOutputIdx >= len(tx.Vout) {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	amount, err := d.decodeDepositOutput(tx.Vout[depositOutputIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode deposit output")
	}

	depositMemo, err := d.decodeDepositMemoOutput(tx.Vout[destinationOutputIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode destination output")
	}

	return &DepositData{
		Amount:      amount,
		DepositMemo: *depositMemo,
	}, nil
}

func (d *DepositDecoder) decodeDepositOutput(out btcjson.Vout) (amount *big.Int, err error) {
	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}
	if !d.helper.ScriptSupported(scriptRaw) {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid deposit output script")
	}

	addresses, err := d.helper.ExtractScriptAddresses(scriptRaw)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}
	if len(addresses) != 1 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "expected exactly one address in deposit output")
	}

	if !d.isBridgeAddress(addresses[0]) {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidReceiverAddress, "deposit output address is not a bridge address")
	}

	if out.Value == 0 {
		return nil, bridgeTypes.ErrInvalidDepositedAmount
	}

	return big.NewInt(utils.ToUnits(out.Value)), nil
}

func (d *DepositDecoder) isBridgeAddress(addr string) bool {
	for _, bridgeAddr := range d.bridgeAddresses {
		if addr == bridgeAddr {
			return true
		}
	}
	return false
}

func (d *DepositDecoder) decodeDepositMemoOutput(out btcjson.Vout) (*DepositMemo, error) {
	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	raw, err := d.helper.RetrieveOpReturnData(scriptRaw)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	depositMemo, err := d.decodeDepositMemo(raw)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	return depositMemo, nil
}

// decodeDepositMemo decodes the deposit memo from raw bytes.
// deposit memo structure:
//
// [chainId][referralId][addressEncodingType][destinationAddress]
//   - chainId: right-padded to 6 bytes, UTF-8
//   - referralId: 2 bytes, big-endian
//   - addressEncodingType: 1 byte
//   - destinationAddress: variable length
func (d *DepositDecoder) decodeDepositMemo(raw []byte) (*DepositMemo, error) {
	if len(raw) <= chainIdLength+referralIdLength+1 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid deposit memo length")
	}

	var depositMemo DepositMemo
	depositMemo.ChainId = string(bytes.TrimRight(raw[:chainIdLength], string(PaddingByte)))
	depositMemo.ReferralId = binary.BigEndian.Uint16(raw[chainIdLength : chainIdLength+referralIdLength])

	encodingTypeByte := raw[chainIdLength+referralIdLength]
	encoder := encoding.GetEncoder(encoding.Type(encodingTypeByte))
	if encoder == nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "unknown address encoding type")
	}

	depositMemo.Address = encoder.Encode(raw[chainIdLength+referralIdLength+1:])

	return &depositMemo, nil
}
