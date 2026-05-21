package service

import (
	"strings"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

func TestLocalSignPartyForProtocolSelectsECDSA(t *testing.T) {
	ecdsaShare := &keygen.LocalPartySaveData{}
	localParty, err := localSignPartyForProtocol(core.Account{}, &secrets.TssShares{Share: ecdsaShare}, 2, signProtocolECDSA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if localParty.Share != ecdsaShare {
		t.Fatal("expected ECDSA share")
	}
	if localParty.FrostShare != nil {
		t.Fatal("did not expect FROST share")
	}
	if localParty.Threshold != 2 {
		t.Fatalf("unexpected threshold: %d", localParty.Threshold)
	}
}

func TestLocalSignPartyForProtocolRejectsMissingECDSA(t *testing.T) {
	_, err := localSignPartyForProtocol(core.Account{}, &secrets.TssShares{}, 2, signProtocolECDSA)
	if err == nil || !strings.Contains(err.Error(), "ECDSA share is required") {
		t.Fatalf("expected missing ECDSA error, got %v", err)
	}
}

func TestLocalSignPartyForProtocolSelectsFROST(t *testing.T) {
	frostShare := &frostkeygen.Config{}
	localParty, err := localSignPartyForProtocol(core.Account{}, &secrets.TssShares{FrostShare: frostShare}, 2, signProtocolFROST)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if localParty.FrostShare != frostShare {
		t.Fatal("expected FROST share")
	}
	if localParty.Share != nil {
		t.Fatal("did not expect ECDSA share")
	}
}

func TestLocalSignPartyForProtocolRejectsMissingFROST(t *testing.T) {
	_, err := localSignPartyForProtocol(core.Account{}, &secrets.TssShares{}, 2, signProtocolFROST)
	if err == nil || !strings.Contains(err.Error(), "FROST share is required") {
		t.Fatalf("expected missing FROST error, got %v", err)
	}
}

func TestValidateSignProtocolRejectsInvalidProtocol(t *testing.T) {
	err := validateSignProtocol("schnorr")
	if err == nil || !strings.Contains(err.Error(), "unsupported signing protocol") {
		t.Fatalf("expected unsupported protocol error, got %v", err)
	}
}
