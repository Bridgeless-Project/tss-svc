package service

import (
	"strings"
	"testing"
)

func TestValidateSignProtocolAcceptsSupportedProtocols(t *testing.T) {
	if err := validateSignProtocol(signProtocolECDSA); err != nil {
		t.Fatalf("unexpected ECDSA error: %v", err)
	}
	if err := validateSignProtocol(signProtocolFROST); err != nil {
		t.Fatalf("unexpected FROST error: %v", err)
	}
}

func TestValidateSignProtocolRejectsInvalidProtocol(t *testing.T) {
	err := validateSignProtocol("schnorr")
	if err == nil || !strings.Contains(err.Error(), "unsupported signing protocol") {
		t.Fatalf("expected unsupported protocol error, got %v", err)
	}
}
