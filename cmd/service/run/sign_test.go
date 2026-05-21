package run

import (
	"testing"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

func TestSelectShareForChain(t *testing.T) {
	ecdsaShare := &keygen.LocalPartySaveData{}
	frostShare := &frostkeygen.Config{}

	tests := []struct {
		name      string
		shares    *secrets.TssShares
		chainType chain.Type
		protocol  int
		wantErr   bool
	}{
		{
			name:      "production chain prefers ecdsa when both shares exist",
			shares:    &secrets.TssShares{Share: ecdsaShare, FrostShare: frostShare},
			chainType: chain.TypeEVM,
			protocol:  tss.ProtocolID_ECDSA,
		},
		{
			name:      "test chain prefers frost when both shares exist",
			shares:    &secrets.TssShares{Share: ecdsaShare, FrostShare: frostShare},
			chainType: chain.TypeOther,
			protocol:  tss.ProtocolID_FROST,
		},
		{
			name:      "ecdsa-only production chain works",
			shares:    &secrets.TssShares{Share: ecdsaShare},
			chainType: chain.TypeBitcoin,
			protocol:  tss.ProtocolID_ECDSA,
		},
		{
			name:      "ecdsa-only test chain fails",
			shares:    &secrets.TssShares{Share: ecdsaShare},
			chainType: chain.TypeOther,
			wantErr:   true,
		},
		{
			name:      "frost-only test chain works",
			shares:    &secrets.TssShares{FrostShare: frostShare},
			chainType: chain.TypeOther,
			protocol:  tss.ProtocolID_FROST,
		},
		{
			name:      "frost-only production chain fails",
			shares:    &secrets.TssShares{FrostShare: frostShare},
			chainType: chain.TypeSolana,
			wantErr:   true,
		},
		{
			name:      "empty shares fail",
			shares:    &secrets.TssShares{},
			chainType: chain.TypeZano,
			wantErr:   true,
		},
		{
			name:      "nil shares fail",
			shares:    nil,
			chainType: chain.TypeOther,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, protocol, err := selectShareForChain(tt.shares, tt.chainType)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if protocol != tt.protocol {
				t.Fatalf("expected protocol %d, got %d", tt.protocol, protocol)
			}
		})
	}
}
