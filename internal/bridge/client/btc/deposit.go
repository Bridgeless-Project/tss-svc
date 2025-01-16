package btc

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/hyle-team/tss-svc/internal/bridge"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"math/big"
	"slices"
	"strings"
)

const (
	defaultDecimals                  = 8
	defaultDepositorAddressOutputIdx = 0

	minOpReturnCodeLen = 3

	dstSeparator   = "-"
	dstParamsCount = 2
	dstAddrIdx     = 0
	dstChainIdIdx  = 1

	dstEthAddrLen  = 42
	dstZanoAddrLen = 71
)

func (p *proxy) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	var (
		depositIdx = id.TxEventId
		dstDataIdx = depositIdx + 1
	)

	tx, err := p.getTransaction(id.TxHash)
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
	block, err := p.chain.Rpc.GetBlockVerbose(blockHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block")
	}
	if tx.Confirmations < p.chain.Confirmations {
		return nil, bridgeTypes.ErrTxNotConfirmed
	}

	if len(tx.Vout) < dstDataIdx+1 || len(tx.Vin) == 0 {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	amount, err := p.parseDepositOutput(tx.Vout[depositIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposit amount")
	}

	addr, chainId, err := p.parseDestinationOutput(tx.Vout[dstDataIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get destination address")
	}

	depositor, err := p.parseSenderAddress(tx.Vin[defaultDepositorAddressOutputIdx])
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

func (p *proxy) parseSenderAddress(in btcjson.Vin) (addr string, err error) {
	prevTx, err := p.getTransaction(in.Txid)
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

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(scriptRaw, p.chain.Params)
	if err != nil {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}
	if len(addrs) == 0 {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "empty sender address")
	}

	return addrs[0].String(), nil
}

func (p *proxy) parseDestinationOutput(out btcjson.Vout) (addr, chainId string, err error) {
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
	if raw[0] != txscript.OP_RETURN {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid script type")
	}
	if len(raw) <= minOpReturnCodeLen {
		return "", errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "destination data missing")
	}

	// Stripping: OP_RETURN OP_PUSH [return data length] (first three bytes)
	return string(raw[3:]), nil
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
	default:
		err = errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid destination address parameter")
	}

	return
}

var supportedScriptTypes = []txscript.ScriptClass{
	txscript.PubKeyHashTy,
	txscript.WitnessV0PubKeyHashTy,
	txscript.WitnessV1TaprootTy,
}

func (p *proxy) parseDepositOutput(out btcjson.Vout) (*big.Int, error) {
	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	stype, addrs, _, err := txscript.ExtractPkScriptAddrs(scriptRaw, p.chain.Params)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}
	if !slices.Contains(supportedScriptTypes, stype) || len(addrs) != 1 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, fmt.Sprintf("unsupported type %s", stype))
	}
	if !p.bridgeAddr(addrs[0]) {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "receiver address is not bridge")
	}
	if out.Value == 0 {
		return nil, bridgeTypes.ErrInvalidDepositedAmount
	}

	return toBigint(out.Value, defaultDecimals), nil
}
