package tss

import (
	"bytes"
	"testing"

	rootTss "github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
)

func TestFrostSignatureSetSignatureValidatesInput(t *testing.T) {
	signature := new(FrostSignature)
	raw := bytes.Repeat([]byte{0x11}, taproot.SignatureLen)

	if err := signature.SetSignature(raw); err != nil {
		t.Fatalf("expected valid FROST signature, got error: %v", err)
	}
	if signature.Format() != rootTss.SignatureFormatSchnorrTaproot {
		t.Fatalf("unexpected signature format: %s", signature.Format())
	}
	if !bytes.Equal(signature.GetSignature(), raw) {
		t.Fatal("stored signature bytes do not match input")
	}
	if signature.GetR() != nil || signature.GetS() != nil || signature.GetM() != nil {
		t.Fatal("FROST signature must not expose ECDSA-shaped components")
	}

	raw[0] = 0xff
	if bytes.Equal(signature.GetSignature(), raw) {
		t.Fatal("signature data aliases caller-owned buffer")
	}
}

func TestFrostSignatureRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{name: "wrong type", input: "not-bytes"},
		{name: "short signature", input: bytes.Repeat([]byte{0x01}, taproot.SignatureLen-1)},
		{name: "long signature", input: bytes.Repeat([]byte{0x01}, taproot.SignatureLen+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := new(FrostSignature)
			if err := signature.SetSignature(tt.input); err == nil {
				t.Fatal("expected malformed FROST signature to be rejected")
			}
		})
	}
}

func TestFrostSignatureGobDecodeRejectsMalformedData(t *testing.T) {
	signature := new(FrostSignature)
	if err := signature.GobDecode(bytes.Repeat([]byte{0x01}, taproot.SignatureLen-1)); err == nil {
		t.Fatal("expected malformed gob payload to be rejected")
	}
	if err := signature.GobDecode(bytes.Repeat([]byte{0x02}, taproot.SignatureLen)); err != nil {
		t.Fatalf("expected valid gob payload, got error: %v", err)
	}
}
