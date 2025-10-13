package client

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/pkg/encoding"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
)

func constructMemoV2(chainId string, referralId uint16, addrEncodingType byte, rawAddr []byte) []byte {
	decodedChainId := append([]byte{byte(len(chainId))}, []byte(chainId)...)
	referralIdBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(referralIdBytes, referralId)

	return append(append(decodedChainId, referralIdBytes...), append([]byte{addrEncodingType}, rawAddr...)...)
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
