package zano

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
)

func Test_ParseDeposit_V1(t *testing.T) {
	tests := map[string]struct {
		prepareMemo func() zanoTypes.ServiceEntry
		expected    DepositMemo
		err         bool
	}{
		"valid memo": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := DepositMemo{
					ChainId: "123",
					Address: "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			expected: DepositMemo{
				ChainId:    "123",
				Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId: 0,
			},
			err: false,
		},
		"invalid memo (missing address)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := map[string]string{
					"dst_net_id": "123",
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			err: true,
		},
		"invalid memo (missing chain id)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := map[string]string{
					"dst_add": "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			err: true,
		},
		"invalid memo (not json)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString([]byte("not a json")),
				}
			},
			err: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			parsed, err := parseDepositMemo(tc.prepareMemo())
			if err != nil {
				if !tc.err {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}
			if tc.err {
				t.Fatal("expected error, got nil")
			}

			if *parsed != tc.expected {
				t.Fatalf("expected memo %v, got %v", tc.expected, *parsed)
			}
		})
	}
}

func Test_ParseDeposit_V2(t *testing.T) {
	tests := map[string]struct {
		prepareMemo func() zanoTypes.ServiceEntry
		expected    DepositMemo
		err         bool
	}{
		"valid memo": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := DepositMemo{
					ChainId:    "123",
					Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
					ReferralId: 123,
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			expected: DepositMemo{
				ChainId:    "123",
				Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId: 123,
			},
			err: false,
		},
		"valid memo (missing ref id)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := DepositMemo{
					ChainId: "123",
					Address: "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			expected: DepositMemo{
				ChainId:    "123",
				Address:    "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				ReferralId: 0,
			},
			err: false,
		},
		"invalid memo (missing address)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := map[string]string{
					"dst_net_id": "123",
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			err: true,
		},
		"invalid memo (missing chain id)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				entry := map[string]string{
					"dst_add": "0xbeefD475A76Ec312502ba7B566a9B4CEA91ab030",
				}
				raw, _ := json.Marshal(entry)
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString(raw),
				}
			},
			err: true,
		},
		"invalid memo (not json)": {
			prepareMemo: func() zanoTypes.ServiceEntry {
				return zanoTypes.ServiceEntry{
					Body: hex.EncodeToString([]byte("not a json")),
				}
			},
			err: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			parsed, err := parseDepositMemo(tc.prepareMemo())
			if err != nil {
				if !tc.err {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}
			if tc.err {
				t.Fatal("expected error, got nil")
			}

			if *parsed != tc.expected {
				t.Fatalf("expected memo %v, got %v", tc.expected, *parsed)
			}
		})
	}
}
