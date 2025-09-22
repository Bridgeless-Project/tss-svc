package client

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/pkg/encoding"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
)

func constructMemo(chainId string, referralId uint16, addrEncodingType byte, rawAddr []byte) []byte {
	// pad chainId to 6 bytes with leading PaddingByte
	paddedChainId := make([]byte, 6)
	copy(paddedChainId, chainId)
	for i := len(chainId); i < 6; i++ {
		paddedChainId[i] = PaddingByte
	}

	referralIdBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(referralIdBytes, referralId)

	return append(append(paddedChainId, referralIdBytes...), append([]byte{addrEncodingType}, rawAddr...)...)
}

func Test_DepositDecoding(t *testing.T) {
	tests := map[string]struct {
		prepareMemo func() []byte
		expected    DepositMemo
		err         bool
	}{
		"valid ETH memo (hex checksum)": {
			prepareMemo: func() []byte {
				rawAddr := common.HexToAddress("0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030").Bytes()
				return constructMemo("123", 123, byte(encoding.TypeHexCheckSum), rawAddr)
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
				return constructMemo("2", 3, byte(encoding.TypeBase58), rawAddr)
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
				return constructMemo("45", 0, byte(encoding.TypeBase64Url), rawAddr)
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
				return constructMemo("101", 65500, byte(encoding.TypeBase58), rawAddr)
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
				return constructMemo("123", 123, 0xFF, rawAddr)
			},
			err: true,
		},
	}

	decoder := &DepositDecoder{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			memo, err := decoder.decodeDepositMemo(tc.prepareMemo())
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
