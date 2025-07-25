package bitcoin

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/xssnick/tonutils-go/address"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/pkg/errors"
)

const (
	defaultDepositorAddressOutputIdx = 0

	dstSeparator   = "#"
	dstParamsCount = 2
	dstAddrIdx     = 0
	dstChainIdIdx  = 1

	dstEthAddrLen  = 42
	dstZanoAddrLen = 71
	dstTonAddrLen  = 48
)

func (c *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	var (
		depositIdx = id.TxNonce
		dstDataIdx = depositIdx + 1
	)

	tx, err := c.GetTransaction(id.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction")
	}

	if tx.BlockHash == "" {
		return nil, bridgeTypes.ErrTxPending
	}
	blockHash, err := chainhash.NewHashFromStr(tx.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode block hash")
	}
	block, err := c.chain.Rpc.Node.GetBlockVerbose(blockHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block")
	}
	if tx.Confirmations < c.chain.Confirmations {
		return nil, bridgeTypes.ErrTxNotConfirmed
	}

	if int64(len(tx.Vout)) < dstDataIdx+1 || len(tx.Vin) == 0 {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	amount, err := c.parseDepositOutput(tx.Vout[depositIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposit amount")
	}

	addr, chainId, err := parseDestinationOutput(tx.Vout[dstDataIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get destination address")
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

func (c *Client) parseSenderAddress(in btcjson.Vin) (addr string, err error) {
	prevTx, err := c.GetTransaction(in.Txid)
	if err != nil {
		return "", errors.Wrap(err, "failed to get previous transaction to identify sender")
	}

	if len(prevTx.Vout) < int(in.Vout)+1 {
		return "", errors.New("sender vout not found")
	}

	scriptRaw, err := hex.DecodeString(prevTx.Vout[in.Vout].ScriptPubKey.Hex)
	if err != nil {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(scriptRaw, c.chain.Params)
	if err != nil {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}
	if len(addrs) == 0 {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "empty sender address")
	}

	return addrs[0].String(), nil
}

func parseDestinationOutput(out btcjson.Vout) (addr, chainId string, err error) {
	if len(out.ScriptPubKey.Hex) == 0 {
		return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "empty destination")
	}

	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	dstData, err := retrieveEncodedDestinationData(scriptRaw)
	if err != nil {
		return addr, chainId, err
	}

	return decodeDestinationData(dstData)
}

func retrieveEncodedDestinationData(raw []byte) (string, error) {
	if !txscript.IsNullData(raw) {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid script type, expected valid OP_RETURN")
	}

	data, err := txscript.PushedData(raw)
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve pushed data from script")
	}
	if len(data) != 1 {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "expected exactly one pushed data item in OP_RETURN script")
	}

	return string(data[0]), nil
}

func decodeDestinationData(data string) (addr, chainId string, err error) {
	params := strings.Split(data, dstSeparator)
	if len(params) != dstParamsCount {
		return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid destination params count")
	}

	addr, chainId = params[dstAddrIdx], params[dstChainIdIdx]

	switch len(addr) {
	case dstEthAddrLen:
		// nothing to decode
		addr = params[0]
	case dstZanoAddrLen:
		// decoding from base58 to get proper user addr representation
		addr = base58.Encode([]byte(params[0]))
	case dstTonAddrLen:
		tonAddr, err := address.ParseAddr(addr)
		if err != nil {
			return addr, chainId, errors.Wrap(bridgeTypes.ErrInvalidReceiverAddress, err.Error())
		}

		addr = tonAddr.String()
	default:
		err = errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid destination address parameter")
	}

	return
}

// TODO: REVIEW SUPPORTED SCRIPTS
var supportedScriptTypes = []txscript.ScriptClass{
	txscript.PubKeyHashTy,
}

func (c *Client) parseDepositOutput(out btcjson.Vout) (*big.Int, error) {
	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	stype, addrs, _, err := txscript.ExtractPkScriptAddrs(scriptRaw, c.chain.Params)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}
	if !slices.Contains(supportedScriptTypes, stype) || len(addrs) != 1 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, fmt.Sprintf("unsupported type %s", stype))
	}
	if !c.IsBridgeAddr(addrs[0]) {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "receiver address is not bridge")
	}

	if out.Value == 0 {
		return nil, bridgeTypes.ErrInvalidDepositedAmount
	}

	return ToAmount(out.Value, Decimals), nil
}
