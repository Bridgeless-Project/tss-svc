package client

import (
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/pkg/encoding"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil/base58"
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

	DestinationToken     *string
	MinDestinationAmount *big.Int
	SwapDeadline         *big.Int
}

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

	depositMemo, err := d.decodeDepositMemo(tx.Vout, destinationOutputIdx)
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

func (d *DepositDecoder) decodeDepositMemo(vouts []btcjson.Vout, memoIdx int) (*DepositMemo, error) {
	memoRaw, err := d.decodeOpReturnOutput(vouts[memoIdx])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode op_return output")
	}

	if chunksCount := d.getMemoChunksCount(memoRaw); chunksCount > 0 {
		chunks, err := d.retrieveMemoChunks(vouts, memoIdx, chunksCount)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve memo chunks")
		}

		memoRaw = append(memoRaw, chunks...)
	}

	if depositMemo, err := d.decodeDepositMemoV3(memoRaw); err == nil {
		return depositMemo, nil
	}

	if depositMemo, err := d.decodeDepositMemoV2(memoRaw); err == nil {
		return depositMemo, nil
	}

	if depositMemo, err := d.decodeDepositMemoV1(memoRaw); err == nil {
		return depositMemo, nil
	}

	return nil, bridgeTypes.ErrInvalidTransactionMemo
}

func (d *DepositDecoder) getMemoChunksCount(rawMemo []byte) int {
	if !d.isMemoV3(rawMemo) {
		return 0
	}

	return int(rawMemo[memoV3ChunksCountIdx])
}

func (d *DepositDecoder) decodeOpReturnOutput(out btcjson.Vout) ([]byte, error) {
	scriptRaw, err := hex.DecodeString(out.ScriptPubKey.Hex)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	raw, err := d.helper.RetrieveOpReturnData(scriptRaw)
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
	}

	return raw, nil
}

func (d *DepositDecoder) retrieveMemoChunks(vouts []btcjson.Vout, memoIdx, chunksCount int) ([]byte, error) {
	if memoIdx+chunksCount >= len(vouts) {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, "not enough outputs for memo chunks")
	}

	chunks := make([]byte, 0)
	for i := 0; i < chunksCount; i++ {
		chunkIdx := memoIdx + 1 + i
		scriptRaw, err := hex.DecodeString(vouts[chunkIdx].ScriptPubKey.Hex)
		if err != nil {
			return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, err.Error())
		}

		chunk, err := d.helper.RetrieveMemoChunkData(scriptRaw)
		if err != nil {
			return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, err.Error())
		}

		chunks = append(chunks, chunk...)
	}

	return chunks, nil
}

// decodeDepositMemoV3 decodes the deposit memo from raw bytes for version 3.
// deposit memo structure:

// [magic][version][chunksCount][isSwap]
//   - magic: 1 byte (0xFF), indicates the start of the memo
//   - version: 1 byte, indicates the version of the memo (0x03 for v3), allows for future extensions
//   - chunksCount: 1 byte, number of following chunk outputs used by this memo
//   - isSwap: 1 byte, indicates if the deposit is a swap (0x00 for false, 0x01 for true)

// Depending on the isSwap value, the structure of the memo will differ.
// If isSwap is false, the v2 structure and decoding logic will be used.
// Otherwise, the following structure will be used for v3 swap deposits (continuing from the isSwap byte):

// [lenChainId][chainId][referralId][addressEncodingType][lenDstAddr][dstAddr][tokenEncodingType][lenDstToken][dstToken][lenMinDstAmount][minDstAmount][lenSwapDeadline][swapDeadline]
//   - lenChainId: 1 byte, length of chainId
//   - chainId: variable length
//   - referralId: 2 bytes, big-endian
//   - addressEncodingType: 1 byte
//   - lenDstAddr: 1 byte, length of destination address
//   - dstAddr: variable length, destination address
//   - tokenEncodingType: 1 byte
//   - lenDstToken: 1 byte, length of destination token address
//   - dstToken: variable length, destination token address
//   - lenMinDstAmount: 1 byte, length of minimum destination amount
//   - minDstAmount: variable length, minimum destination amount (big-endian)
//   - lenSwapDeadline: 1 byte, length of swap deadline
//   - swapDeadline: variable length, swap deadline timestamp in seconds (big-endian)
func (d *DepositDecoder) decodeDepositMemoV3(raw []byte) (*DepositMemo, error) {
	if !d.isMemoV3(raw) {
		return nil, bridgeTypes.ErrInvalidTransactionMemo
	}

	memo := raw[memoV3HeaderLength:]
	switch raw[memoV3SwapFlagIdx] {
	case memoV3NoSwap:
		// decode v2-compatible deposit memo
		return d.decodeDepositMemoV2(memo)
	case memoV3Swap:
		// continue below
	default:
		return nil, bridgeTypes.ErrInvalidTransactionMemo
	}

	// decode v3 swap deposit memo ensuring length checks for each field
	memoReader := NewMemoReader(memo)
	chainId, err := memoReader.ReadLenBytes()
	if err != nil {
		return nil, err
	}

	referralId, err := memoReader.ReadBytes(memoReferralIdLength)
	if err != nil {
		return nil, err
	}

	encodingTypeByte, err := memoReader.ReadByte()
	if err != nil {
		return nil, err
	}
	encoder := encoding.GetEncoder(encoding.Type(encodingTypeByte))
	if encoder == nil {
		return nil, bridgeTypes.ErrInvalidTransactionMemo
	}

	dstAddr, err := memoReader.ReadLenBytes()
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, err.Error())
	}

	tokenEncodingTypeByte, err := memoReader.ReadByte()
	if err != nil {
		return nil, err
	}
	tokenEncoder := encoding.GetEncoder(encoding.Type(tokenEncodingTypeByte))
	if tokenEncoder == nil {
		return nil, bridgeTypes.ErrInvalidTransactionMemo
	}

	dstToken, err := memoReader.ReadLenBytes()
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, err.Error())
	}

	minDstAmount, err := memoReader.ReadLenBytes()
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, err.Error())
	}

	swapDeadline, err := memoReader.ReadLenBytes()
	if err != nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidTransactionMemo, err.Error())
	}

	return &DepositMemo{
		ChainId:              string(chainId),
		ReferralId:           binary.BigEndian.Uint16(referralId),
		Address:              encoder.Encode(dstAddr),
		DestinationToken:     new(tokenEncoder.Encode(dstToken)),
		MinDestinationAmount: new(big.Int).SetBytes(minDstAmount),
		SwapDeadline:         new(big.Int).SetBytes(swapDeadline),
	}, nil
}

// decodeDepositMemoV2 decodes the deposit memo from raw bytes.
// deposit memo structure:
//
// [lenChainId][chainId][referralId][addressEncodingType][destinationAddress]
//   - lenChainId: 1 byte, length of chainId
//   - chainId: variable length
//   - referralId: 2 bytes, big-endian
//   - addressEncodingType: 1 byte
//   - destinationAddress: variable length
func (d *DepositDecoder) decodeDepositMemoV2(raw []byte) (*DepositMemo, error) {
	if len(raw) == 0 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "empty deposit memo")
	}

	chainIdLength := int(raw[0])
	if len(raw) <= 1+chainIdLength+memoReferralIdLength+1 {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "invalid deposit memo length")
	}
	chainIdEndIdx := 1 + chainIdLength

	var depositMemo DepositMemo
	depositMemo.ChainId = string(raw[1:chainIdEndIdx])
	depositMemo.ReferralId = binary.BigEndian.Uint16(raw[chainIdEndIdx : chainIdEndIdx+memoReferralIdLength])

	encodingTypeByte := raw[chainIdEndIdx+memoReferralIdLength]
	encoder := encoding.GetEncoder(encoding.Type(encodingTypeByte))
	if encoder == nil {
		return nil, errors.Wrap(bridgeTypes.ErrInvalidScriptPubKey, "unknown address encoding type")
	}

	depositMemo.Address = encoder.Encode(raw[chainIdEndIdx+memoReferralIdLength+1:])

	return &depositMemo, nil
}

const (
	memoV1DstSeparator   = "#"
	memoV1DstParamsCount = 2
	memoV1DstAddrIdx     = 0
	memoV1DstChainIdIdx  = 1

	memoV1DstZanoAddrLen = 71
)

func (d *DepositDecoder) decodeDepositMemoV1(raw []byte) (*DepositMemo, error) {
	parts := strings.Split(string(raw), memoV1DstSeparator)
	if len(parts) < memoV1DstParamsCount {
		return nil, errors.New("invalid destination parameters")
	}
	if len(parts) > memoV1DstParamsCount {
		// try concatenating all but the last parts in case the raw bytes sequence in a string contains the separator
		parts = []string{
			strings.Join(parts[:len(parts)-1], memoV1DstSeparator),
			parts[len(parts)-1],
		}
	}

	addr, chainId := parts[memoV1DstAddrIdx], parts[memoV1DstChainIdIdx]
	if len(addr) == 0 || len(chainId) == 0 {
		return nil, errors.New("invalid destination parameters")
	}

	if len(addr) == memoV1DstZanoAddrLen {
		addr = base58.Encode([]byte(addr))
	}

	return &DepositMemo{
		Address:    addr,
		ChainId:    chainId,
		ReferralId: 0,
	}, nil
}

func (d *DepositDecoder) isMemoV3(rawMemo []byte) bool {
	return len(rawMemo) >= memoV3HeaderLength &&
		rawMemo[memoMagicByteIdx] == memoMagicByte &&
		rawMemo[memoV3VersionIdx] == memoV3Version
}
