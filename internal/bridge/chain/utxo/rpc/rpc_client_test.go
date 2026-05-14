package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Asuth(t *testing.T) {
	tests := []struct {
		name         string
		settings     Settings
		expectErr    bool
		checkUnspent bool
	}{
		{
			name: "Valid Regtest Connection",
			settings: Settings{
				Host:     "127.0.0.1:18443",
				User:     "user",
				Password: "password",
				Chain:    "regtest",
			},
			expectErr:    false,
			checkUnspent: true,
		},
		{
			name: "Invalid Regtest Connection",
			settings: Settings{
				Host:     "127.0.0.1:18443",
				User:     "wrong_user",
				Password: "wrong_password",
				Chain:    "regtest",
			},
			expectErr:    true,
			checkUnspent: true,
		},
		{
			name: "Valid Testnet Connection",
			settings: Settings{
				Host:     "18.215.149.26:30866/wallet/tssfl",
				User:     "bitcoin",
				Password: "bitcoin",
				Chain:    "testnet",
			},
			expectErr:    false,
			checkUnspent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.settings)

			if tt.expectErr {
				unspent, err := client.ListUnspent(0)
				assert.Error(t, err, "Expected an error for case: %s", tt.name)
				assert.Nil(t, unspent)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, client)

			if tt.checkUnspent {
				unspent, err := client.ListUnspent(0)
				assert.NoError(t, err)
				assert.NotNil(t, unspent)

				if len(unspent) > 0 {
					t.Logf("[%s] Found %d UTXOs. Amount: %f", tt.name, len(unspent), unspent[0].Amount)
				} else {
					t.Logf("[%s] Success: Wallet is connected but empty.", tt.name)
				}
			}
		})
	}
}
