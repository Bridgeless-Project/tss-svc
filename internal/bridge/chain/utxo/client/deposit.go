package client

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/gagliardetto/solana-go"
	"github.com/pkg/errors"
)

const (
	dstSeparator   = "#"
	dstParamsCount = 2
	dstAddrIdx     = 0
	dstChainIdIdx  = 1

	dstEthAddrLen  = 42
	dstZanoAddrLen = 71
	dstTonAddrLen  = 48

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

	addr, chainId, amount, err := c.depositDecoder.Decode(tx, id.TxNonce)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode deposit data")
	}

	depositor, err := c.parseSenderAddress(tx.Vin[defaultDepositorAddressOutputIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get depositor")
	}

	return &db.DepositData{
		DepositIdentifier:  id,
		DestinationChainId: chainId,
		DestinationAddress: addr,
		SourceAddress:      depositor,
		DepositAmount:      amount,
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

func NewDepositDecoder(helper helper.UtxoHelper, bridgeAddresses []string) *DepositDecoder {
	return &DepositDecoder{
		helper:          helper,
		bridgeAddresses: bridgeAddresses,
	}
}

func (d *DepositDecoder) Decode(tx *btcjson.TxRawResult, depositIdx int64) (addr, chainId string, amount *big.Int, err error) {
	if depositIdx < 0 {
		return "", "", nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionData, "invalid deposit index")
	}
	var (
		depositOutputIdx     = int(depositIdx)
		destinationOutputIdx = depositOutputIdx + 1
	)

	if depositOutputIdx < 0 || destinationOutputIdx >= len(tx.Vout) {
		return "", "", nil, bridgeTypes.ErrDepositNotFound
	}

	amount, err = d.decodeDepositOutput(tx.Vout[depositOutputIdx])
	if err != nil {
		return "", "", nil, errors.Wrap(err, "failed to decode deposit output")
	}

	addr, chainId, err = d.decodeDestinationOutput(tx.Vout[destinationOutputIdx])
	if err != nil {
		return "", "", nil, errors.Wrap(err, "failed to decode destination output")
	}

	return
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

func (d *DepositDecoder) decodeDestinationOutput(out btcjson.Vout) (addr, chainId string, err error) {
	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	raw, err := d.helper.RetrieveOpReturnData(scriptRaw)
	if err != nil {
		return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	addr, chainId, err = decodeDestinationData(raw)
	if err != nil {
		return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	return
}

func decodeDestinationData(raw string) (addr, chainId string, err error) {
	parts := strings.Split(raw, dstSeparator)
	if len(parts) != dstParamsCount {
		return addr, chainId, errors.New("invalid destination parameters")
	}

	addr, chainId = parts[dstAddrIdx], parts[dstChainIdIdx]

	switch len(addr) {
	case dstEthAddrLen, dstTonAddrLen:
		return
	case dstZanoAddrLen:
		addr = base58.Encode([]byte(addr))
		return
	default:
		// Solana address is not fixed-length, so just try parsing it
		_, err = solana.PublicKeyFromBase58(addr)
		if err != nil {
			err = errors.New("invalid destination address parameter")
			return
		}
	}
	return
}
