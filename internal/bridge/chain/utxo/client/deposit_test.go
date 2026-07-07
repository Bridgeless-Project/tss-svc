package client

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	stdhex "encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	bchutxohelper "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/bch"
	btcutxohelper "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/btc"
	"github.com/Bridgeless-Project/tss-svc/pkg/encoding"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	btcscript "github.com/btcsuite/btcd/txscript"
	"github.com/ethereum/go-ethereum/common"
	bchcfg "github.com/gcash/bchd/chaincfg"
	bchscript "github.com/gcash/bchd/txscript"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
)

func constructMemoV2(chainId string, referralId uint16, addrEncodingType byte, rawAddr []byte) []byte {
	decodedChainId := append([]byte{byte(len(chainId))}, []byte(chainId)...)
	referralIdBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(referralIdBytes, referralId)

	return append(append(decodedChainId, referralIdBytes...), append([]byte{addrEncodingType}, rawAddr...)...)
}

func constructMemoV3NoSwap(chainId string, referralId uint16, addrEncodingType byte, rawAddr []byte) []byte {
	memo := []byte{memoMagicByte, memoV3Version, 0x00, memoV3NoSwap}
	return append(memo, constructMemoV2(chainId, referralId, addrEncodingType, rawAddr)...)
}

func constructMemoV3Swap(
	chainId string,
	referralId uint16,
	addrEncodingType byte,
	rawAddr []byte,
	tokenEncodingType byte,
	rawDstToken []byte,
	minDstAmount *big.Int,
	swapDeadline *big.Int,
) []byte {
	referralIdBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(referralIdBytes, referralId)

	minDstAmountBytes := minDstAmount.Bytes()
	swapDeadlineBytes := swapDeadline.Bytes()

	memo := []byte{memoMagicByte, memoV3Version, 0x00, memoV3Swap}
	memo = append(memo, byte(len(chainId)))
	memo = append(memo, []byte(chainId)...)
	memo = append(memo, referralIdBytes...)
	memo = append(memo, addrEncodingType)
	memo = append(memo, byte(len(rawAddr)))
	memo = append(memo, rawAddr...)
	memo = append(memo, tokenEncodingType)
	memo = append(memo, byte(len(rawDstToken)))
	memo = append(memo, rawDstToken...)
	memo = append(memo, byte(len(minDstAmountBytes)))
	memo = append(memo, minDstAmountBytes...)
	memo = append(memo, byte(len(swapDeadlineBytes)))
	memo = append(memo, swapDeadlineBytes...)

	return memo
}

func constructOpReturnScript(data []byte) string {
	script, err := btcscript.NewScriptBuilder().AddOp(btcscript.OP_RETURN).AddData(data).Script()
	if err != nil {
		panic(err)
	}

	return stdhex.EncodeToString(script)
}

func constructP2WSHChunkScript(chunk []byte) string {
	raw := make([]byte, 32)
	copy(raw, chunk)

	script := append([]byte{btcscript.OP_0, btcscript.OP_DATA_32}, raw...)
	return stdhex.EncodeToString(script)
}

func constructP2PKHChunkScript(chunk []byte) string {
	raw := make([]byte, 20)
	copy(raw, chunk)

	script, err := bchscript.NewScriptBuilder().
		AddOp(bchscript.OP_DUP).
		AddOp(bchscript.OP_HASH160).
		AddData(raw).
		AddOp(bchscript.OP_EQUALVERIFY).
		AddOp(bchscript.OP_CHECKSIG).
		Script()
	if err != nil {
		panic(err)
	}

	return stdhex.EncodeToString(script)
}

type memoOutputType int

const (
	btcMemoOutput memoOutputType = iota
	bchMemoOutput
)

func (t memoOutputType) opReturnSize() int {
	switch t {
	case btcMemoOutput:
		return 80
	case bchMemoOutput:
		return 223
	default:
		panic("unknown memo output type")
	}
}

func (t memoOutputType) chunkSize() int {
	switch t {
	case btcMemoOutput:
		return 32
	case bchMemoOutput:
		return 20
	default:
		panic("unknown memo output type")
	}
}

func (t memoOutputType) constructChunkScript(chunk []byte) string {
	switch t {
	case btcMemoOutput:
		return constructP2WSHChunkScript(chunk)
	case bchMemoOutput:
		return constructP2PKHChunkScript(chunk)
	default:
		panic("unknown memo output type")
	}
}

func constructMemoVouts(memo []byte, outputType memoOutputType) []btcjson.Vout {
	memo = append([]byte{}, memo...)
	opReturnSize := outputType.opReturnSize()
	chunkSize := outputType.chunkSize()

	if len(memo) <= opReturnSize {
		memo[memoV3ChunksCountIdx] = 0
		return []btcjson.Vout{
			{
				ScriptPubKey: btcjson.ScriptPubKeyResult{
					Hex: constructOpReturnScript(memo),
				},
			},
		}
	}

	chunksCount := (len(memo) - opReturnSize + chunkSize - 1) / chunkSize
	if chunksCount > 255 {
		panic("too many memo chunks")
	}
	memo[memoV3ChunksCountIdx] = byte(chunksCount)

	vouts := []btcjson.Vout{
		{
			ScriptPubKey: btcjson.ScriptPubKeyResult{
				Hex: constructOpReturnScript(memo[:opReturnSize]),
			},
		},
	}

	chunkPayload := memo[opReturnSize:]
	for len(chunkPayload) > 0 {
		currentChunkSize := chunkSize
		if len(chunkPayload) < currentChunkSize {
			currentChunkSize = len(chunkPayload)
		}

		chunk := chunkPayload[:currentChunkSize]
		chunkPayload = chunkPayload[currentChunkSize:]

		vouts = append(vouts, btcjson.Vout{
			ScriptPubKey: btcjson.ScriptPubKeyResult{
				Hex: outputType.constructChunkScript(chunk),
			},
		})
	}

	return vouts
}

func Test_DecodeDepositMemo_V1(t *testing.T) {
	tests := map[string]struct {
		prepareMemo func() []byte
		expected    DepositMemo
		err         bool
	}{
		"valid memo (ETH)": {
			prepareMemo: func() []byte {
				return []byte("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030#123")
			},
			expected: DepositMemo{
				ChainId:    "123",
				Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId: 0,
			},
		},
		"valid memo (ZANO)": {
			prepareMemo: func() []byte {
				raw, _ := base58.Decode("ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q")
				return []byte(string(raw) + "#2")
			},
			expected: DepositMemo{
				ChainId:    "2",
				Address:    "ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q",
				ReferralId: 0,
			},
		},
		"valid memo (ZANO which produces # in base58)": {
			prepareMemo: func() []byte {
				raw, _ := base58.Decode("ZxDVeKjCvceATxJ75a6BULddbcytgxHweGjRPqioF9pgF9YSUkFe7fo56WgGr6izuPjg74p4iJvPeY4xNntuoerK1WKNMJQoZ")
				return []byte(string(raw) + "#2")
			},
			expected: DepositMemo{
				ChainId: "2",
				Address: "ZxDVeKjCvceATxJ75a6BULddbcytgxHweGjRPqioF9pgF9YSUkFe7fo56WgGr6izuPjg74p4iJvPeY4xNntuoerK1WKNMJQoZ",
			},
		},
		"invalid memo (missing separator)": {
			prepareMemo: func() []byte {
				return []byte("1230xbeefD475A76Ec312502ba7B566a9B4CEA91ab030")
			},
			err: true,
		},
	}

	decoder := &DepositDecoder{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			memo, err := decoder.decodeDepositMemoV1(tc.prepareMemo())
			if err != nil {
				if !tc.err {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.err {
				t.Fatal("expected error but got none")
			}

			if *memo != tc.expected {
				t.Fatalf("expected memo %v, got %v", tc.expected, *memo)
			}
		})
	}
}

func Test_DecodeDepositMemo_V2(t *testing.T) {
	tests := map[string]struct {
		prepareMemo func() []byte
		expected    DepositMemo
		err         bool
	}{
		"valid ETH memo (hex checksum)": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				return constructMemoV2("123", 123, byte(encoding.TypeHexCheckSum), rawAddr)
			},
			expected: DepositMemo{
				ChainId:    "123",
				Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId: 123,
			},
		},
		"valid ZANO memo (base58)": {
			prepareMemo: func() []byte {
				rawAddr, _ := base58.Decode("ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q")
				return constructMemoV2("2", 3, byte(encoding.TypeBase58), rawAddr)
			},
			expected: DepositMemo{
				ChainId:    "2",
				Address:    "ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q",
				ReferralId: 3,
			},
		},
		"valid TON memo (base64url)": {
			prepareMemo: func() []byte {
				rawAddr, _ := base64.URLEncoding.DecodeString("EQDKbjIcfM6ezt8KjKJJLshZJJSqX7XOA4ff-W72r5gqPrHF")
				return constructMemoV2("45", 0, byte(encoding.TypeBase64Url), rawAddr)
			},
			expected: DepositMemo{
				ChainId:    "45",
				Address:    "EQDKbjIcfM6ezt8KjKJJLshZJJSqX7XOA4ff-W72r5gqPrHF",
				ReferralId: 0,
			},
		},
		"valid Solana memo (base58)": {
			prepareMemo: func() []byte {
				rawAddr, _ := base58.Decode("14grJpemFaf88c8tiVb77W7TYg2W3ir6pfkKz3YjhhZ5")
				return constructMemoV2("101", 65500, byte(encoding.TypeBase58), rawAddr)
			},
			expected: DepositMemo{
				ChainId:    "101",
				Address:    "14grJpemFaf88c8tiVb77W7TYg2W3ir6pfkKz3YjhhZ5",
				ReferralId: 65500,
			},
		},
		"invalid memo (too short)": {
			prepareMemo: func() []byte {
				return []byte{0x01, 0x02, 0x03}
			},
			err: true,
		},
		"invalid encoding type": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				return constructMemoV2("123", 123, 0xFF, rawAddr)
			},
			err: true,
		},
	}

	decoder := &DepositDecoder{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			memo, err := decoder.decodeDepositMemoV2(tc.prepareMemo())
			if err != nil {
				if !tc.err {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.err {
				t.Fatal("expected error but got none")
			}

			if *memo != tc.expected {
				t.Fatalf("expected memo %v, got %v", tc.expected, *memo)
			}
		})
	}
}

func Test_DecodeDepositMemo_V3(t *testing.T) {
	tests := map[string]struct {
		prepareMemo func() []byte
		expected    DepositMemo
		err         bool
	}{
		"valid non-swap ETH memo (v2-compatible)": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				return constructMemoV3NoSwap("123", 123, byte(encoding.TypeHexCheckSum), rawAddr)
			},
			expected: DepositMemo{
				ChainId:    "123",
				Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId: 123,
			},
		},
		"valid non-swap ZANO memo (v2-compatible)": {
			prepareMemo: func() []byte {
				rawAddr, _ := base58.Decode("ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q")
				return constructMemoV3NoSwap("2", 3, byte(encoding.TypeBase58), rawAddr)
			},
			expected: DepositMemo{
				ChainId:    "2",
				Address:    "ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q",
				ReferralId: 3,
			},
		},
		"valid swap ETH memo": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				rawToken := common.HexToAddress("0x0000000000000000000000000000000000000000").Bytes()
				return constructMemoV3Swap(
					"123",
					123,
					byte(encoding.TypeHexCheckSum),
					rawAddr,
					byte(encoding.TypeHexCheckSum),
					rawToken,
					big.NewInt(100000000000),
					big.NewInt(1760000000),
				)
			},
			expected: DepositMemo{
				ChainId:              "123",
				Address:              "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId:           123,
				DestinationToken:     new("0x0000000000000000000000000000000000000000"),
				MinDestinationAmount: big.NewInt(100000000000),
				SwapDeadline:         big.NewInt(1760000000),
			},
		},
		"valid swap ZANO memo": {
			prepareMemo: func() []byte {
				rawAddr, _ := base58.Decode("ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q")
				rawToken, _ := stdhex.DecodeString("eece1a1c0cd74e2b7bae998e3fa9247f8162e1604f843960eb487251bfd8462b")
				return constructMemoV3Swap(
					"2",
					3,
					byte(encoding.TypeBase58),
					rawAddr,
					byte(encoding.TypeHexNoPrefix),
					rawToken,
					big.NewInt(100000000000),
					big.NewInt(1760000000),
				)
			},
			expected: DepositMemo{
				ChainId:              "2",
				Address:              "ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q",
				ReferralId:           3,
				DestinationToken:     new("eece1a1c0cd74e2b7bae998e3fa9247f8162e1604f843960eb487251bfd8462b"),
				MinDestinationAmount: big.NewInt(100000000000),
				SwapDeadline:         big.NewInt(1760000000),
			},
		},
		"invalid memo (too short)": {
			prepareMemo: func() []byte {
				return []byte{memoMagicByte, memoV3Version}
			},
			err: true,
		},
		"invalid swap flag": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				memo := constructMemoV3NoSwap("123", 123, byte(encoding.TypeHexCheckSum), rawAddr)
				memo[3] = 0xFF
				return memo
			},
			err: true,
		},
		"valid swap memo with trailing padding": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				memo := constructMemoV3Swap(
					"123",
					123,
					byte(encoding.TypeHexCheckSum),
					rawAddr,
					byte(encoding.TypeHexCheckSum),
					common.HexToAddress("0x0000000000000000000000000000000000000000").Bytes(),
					big.NewInt(100000000000),
					big.NewInt(1760000000),
				)
				return append(memo, bytes.Repeat([]byte{0x00}, 10)...)
			},
			expected: DepositMemo{
				ChainId:              "123",
				Address:              "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId:           123,
				DestinationToken:     new("0x0000000000000000000000000000000000000000"),
				MinDestinationAmount: big.NewInt(100000000000),
				SwapDeadline:         big.NewInt(1760000000),
			},
		},
		"invalid swap memo (zero-length field)": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				memo := constructMemoV3Swap(
					"123",
					123,
					byte(encoding.TypeHexCheckSum),
					rawAddr,
					byte(encoding.TypeHexCheckSum),
					common.HexToAddress("0x0000000000000000000000000000000000000000").Bytes(),
					big.NewInt(100000000000),
					big.NewInt(1760000000),
				)
				memo[memoV3HeaderLength] = 0x00 // lenChainId
				return memo
			},
			err: true,
		},
	}

	decoder := &DepositDecoder{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			memo, err := decoder.decodeDepositMemoV3(tc.prepareMemo())
			if err != nil {
				if !tc.err {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.err {
				t.Fatal("expected error but got none")
			}

			require.Equal(t, tc.expected, *memo)
		})
	}
}

func Test_DecodeDepositMemo_ChunkedV3(t *testing.T) {
	ethRawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
	ethRawToken := common.HexToAddress("0x0000000000000000000000000000000000000000").Bytes()
	ethMemo := constructMemoV3Swap(
		"123",
		123,
		byte(encoding.TypeHexCheckSum),
		ethRawAddr,
		byte(encoding.TypeHexCheckSum),
		ethRawToken,
		big.NewInt(100000000000),
		big.NewInt(1760000000),
	)
	ethExpected := DepositMemo{
		ChainId:              "123",
		Address:              "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
		ReferralId:           123,
		DestinationToken:     new("0x0000000000000000000000000000000000000000"),
		MinDestinationAmount: big.NewInt(100000000000),
		SwapDeadline:         big.NewInt(1760000000),
	}

	longEthChainId := strings.Repeat("1", 90)
	longEthMemo := constructMemoV3Swap(
		longEthChainId,
		123,
		byte(encoding.TypeHexCheckSum),
		ethRawAddr,
		byte(encoding.TypeHexCheckSum),
		ethRawToken,
		big.NewInt(100000000000),
		big.NewInt(1760000000),
	)
	longEthExpected := DepositMemo{
		ChainId:              longEthChainId,
		Address:              "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
		ReferralId:           123,
		DestinationToken:     new("0x0000000000000000000000000000000000000000"),
		MinDestinationAmount: big.NewInt(100000000000),
		SwapDeadline:         big.NewInt(1760000000),
	}

	zanoRawAddr, _ := base58.Decode("ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q")
	zanoRawToken, _ := stdhex.DecodeString("eece1a1c0cd74e2b7bae998e3fa9247f8162e1604f843960eb487251bfd8462b")
	longZanoChainId := strings.Repeat("2", 180)
	longZanoMemo := constructMemoV3Swap(
		longZanoChainId,
		3,
		byte(encoding.TypeBase58),
		zanoRawAddr,
		byte(encoding.TypeHexNoPrefix),
		zanoRawToken,
		big.NewInt(100000000000),
		big.NewInt(1760000000),
	)
	longZanoExpected := DepositMemo{
		ChainId:              longZanoChainId,
		Address:              "ZxCQtdbr6YPHEWr5Yucy2wTT87yKPwBMHGfJg8RVXg2m3bCDzgWjtbxR7TtGPgDxhWNrauwyPAKEyDdknkBG3Rit1Do9rXG1q",
		ReferralId:           3,
		DestinationToken:     new("eece1a1c0cd74e2b7bae998e3fa9247f8162e1604f843960eb487251bfd8462b"),
		MinDestinationAmount: big.NewInt(100000000000),
		SwapDeadline:         big.NewInt(1760000000),
	}

	tests := map[string]struct {
		memo       []byte
		outputType memoOutputType
		decoder    *DepositDecoder
		chunked    bool
		expected   DepositMemo
	}{
		"BTC 0 chunks": {
			memo:       ethMemo,
			outputType: btcMemoOutput,
			decoder: &DepositDecoder{
				helper: btcutxohelper.NewHelper(&chaincfg.MainNetParams),
			},
			chunked:  false,
			expected: ethExpected,
		},
		"BCH 0 chunks": {
			memo:       ethMemo,
			outputType: bchMemoOutput,
			decoder: &DepositDecoder{
				helper: bchutxohelper.NewHelper(&bchcfg.MainNetParams),
			},
			chunked:  false,
			expected: ethExpected,
		},
		"BTC n chunks": {
			memo:       longEthMemo,
			outputType: btcMemoOutput,
			decoder: &DepositDecoder{
				helper: btcutxohelper.NewHelper(&chaincfg.MainNetParams),
			},
			chunked:  true,
			expected: longEthExpected,
		},
		"BCH n chunks": {
			memo:       longZanoMemo,
			outputType: bchMemoOutput,
			decoder: &DepositDecoder{
				helper: bchutxohelper.NewHelper(&bchcfg.MainNetParams),
			},
			chunked:  true,
			expected: longZanoExpected,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			vouts := constructMemoVouts(tc.memo, tc.outputType)
			fmt.Println(vouts)
			if tc.chunked {
				require.Greater(t, len(vouts), 1)
			} else {
				require.Len(t, vouts, 1)
			}

			depositMemo, err := tc.decoder.decodeDepositMemo(
				vouts,
				0,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, *depositMemo)
		})
	}
}
