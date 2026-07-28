package resharing

import (
	"testing"

	frostTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
)

func TestSetECDSANewPubKeyRejectsUnsupportedShares(t *testing.T) {
	tests := []struct {
		name string
		set  func(*resharingTypes.State)
	}{
		{name: "nil share", set: func(state *resharingTypes.State) {}},
		{name: "frost share", set: func(state *resharingTypes.State) {
			state.NewShare = frostTss.NewFrostShare()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &resharingTypes.State{}
			tt.set(state)
			if err := setECDSANewPubKey(state); err == nil {
				t.Fatal("expected unsupported resharing share to be rejected")
			}
		})
	}
}
