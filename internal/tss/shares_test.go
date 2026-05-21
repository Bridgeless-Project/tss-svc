package tss

import (
	"strings"
	"testing"

	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
)

func TestECDSAShareFromProtocolAcceptsECDSA(t *testing.T) {
	expected := &keygen.LocalPartySaveData{}

	actual, err := ECDSAShareFromProtocol(expected, ProtocolID_ECDSA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual != expected {
		t.Fatal("expected original ECDSA share pointer")
	}
}

func TestECDSAShareFromProtocolRejectsFROST(t *testing.T) {
	_, err := ECDSAShareFromProtocol(&keygen.LocalPartySaveData{}, ProtocolID_FROST)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "resharing supports only ECDSA") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestECDSAShareFromProtocolRejectsUnexpectedShareType(t *testing.T) {
	_, err := ECDSAShareFromProtocol("not-a-share", ProtocolID_ECDSA)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "expected ECDSA TSS share") {
		t.Fatalf("unexpected error: %v", err)
	}
}
