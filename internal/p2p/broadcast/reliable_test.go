package broadcast_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	tsscore "github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsaTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frostTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	tsslib "github.com/bnb-chain/tss-lib/v3/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
)

func TestRoundMessageEncodeDecodeECDSASignatures(t *testing.T) {
	signature := new(ecdsaTss.EcdsaSignature)
	if err := signature.SetSignature(&tsslib.SignatureData{
		Signature:         []byte{0x01},
		SignatureRecovery: []byte{0x02},
		R:                 bytes.Repeat([]byte{0x03}, 32),
		S:                 bytes.Repeat([]byte{0x04}, 32),
		M:                 bytes.Repeat([]byte{0x05}, 32),
	}); err != nil {
		t.Fatalf("failed to set signature: %v", err)
	}

	decodedSignature := encodeDecodeSignature(t, signature)
	if !bytes.Equal(decodedSignature.GetSignature(), signature.GetSignature()) {
		t.Fatal("decoded signature bytes do not match")
	}
	if !bytes.Equal(decodedSignature.GetR(), signature.GetR()) {
		t.Fatal("decoded R component does not match")
	}
	if !bytes.Equal(decodedSignature.GetS(), signature.GetS()) {
		t.Fatal("decoded S component does not match")
	}
	if !bytes.Equal(decodedSignature.GetSignatureRecovery(), signature.GetSignatureRecovery()) {
		t.Fatal("decoded recovery component does not match")
	}
}

func TestRoundMessageEncodeDecodeFROSTSignatures(t *testing.T) {
	signature := new(frostTss.FrostSignature)
	if err := signature.SetSignature(bytes.Repeat([]byte{0x07}, 64)); err != nil {
		t.Fatalf("failed to set signature: %v", err)
	}

	decodedSignature := encodeDecodeSignature(t, signature)
	if !bytes.Equal(decodedSignature.GetSignature(), signature.GetSignature()) {
		t.Fatal("decoded signature bytes do not match")
	}
	if decodedSignature.GetR() != nil || decodedSignature.GetS() != nil || decodedSignature.GetSignatureRecovery() != nil {
		t.Fatal("FROST signature must not expose ECDSA-shaped components")
	}
}

func TestReliableBroadcastReturnsInitialEncodeError(t *testing.T) {
	account := newTestAccount(t)

	broadcaster := broadcast.NewReliable[unencodablePayload](
		"test-session",
		nil,
		*account,
		0,
		p2p.RequestType_RT_SIGNATURE_DISTRIBUTION,
		logan.New(),
	)

	payload := unencodablePayload{Fn: func() {}}
	err := broadcaster.Broadcast(&payload)
	if err == nil {
		t.Fatal("expected broadcast to return encode error")
	}
	if !strings.Contains(err.Error(), "failed to broadcast initial round message") {
		t.Fatalf("expected initial broadcast context, got %q", err)
	}
	if !strings.Contains(err.Error(), "failed to encode round message") {
		t.Fatalf("expected encode context, got %q", err)
	}
}

func TestReliableBroadcastRejectsNegativeRound(t *testing.T) {
	account := newTestAccount(t)
	broadcaster := broadcast.NewReliable[tsscore.Signatures](
		"test-session",
		nil,
		*account,
		0,
		p2p.RequestType_RT_SIGNATURE_DISTRIBUTION,
		logan.New(),
	)

	valid := broadcaster.EnsureValid(broadcast.ReliableBroadcastMsg[tsscore.Signatures]{
		Sender: account.CosmosAddress(),
		Msg: broadcast.RoundMessage[tsscore.Signatures]{
			SessionId: "test-session",
			Round:     -1,
		},
	})
	if valid {
		t.Fatal("expected negative round to be rejected")
	}
}

func TestReliableBroadcastRejectsEmptySignatureChain(t *testing.T) {
	account := newTestAccount(t)
	broadcaster := broadcast.NewReliable[tsscore.Signatures](
		"test-session",
		nil,
		*account,
		0,
		p2p.RequestType_RT_SIGNATURE_DISTRIBUTION,
		logan.New(),
	)

	valid := broadcaster.EnsureValid(broadcast.ReliableBroadcastMsg[tsscore.Signatures]{
		Sender: account.CosmosAddress(),
		Msg: broadcast.RoundMessage[tsscore.Signatures]{
			SessionId:  "test-session",
			Round:      0,
			Signatures: nil,
		},
	})
	if valid {
		t.Fatal("expected empty signature chain to be rejected")
	}
}

func encodeDecodeSignature(t *testing.T, signature tsscore.SignatureData) tsscore.SignatureData {
	t.Helper()

	signatures := &tsscore.Signatures{Data: []tsscore.SignatureData{signature}}
	msg := broadcast.RoundMessage[tsscore.Signatures]{
		Value:     signatures,
		SessionId: "SIGN_test_2",
		Round:     0,
	}

	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("failed to encode round message: %v", err)
	}

	decoded, err := broadcast.DecodeRoundMessage[tsscore.Signatures](encoded)
	if err != nil {
		t.Fatalf("failed to decode round message: %v", err)
	}
	if decoded.Value == nil {
		t.Fatal("decoded value is nil")
	}
	if len(decoded.Value.Data) != 1 {
		t.Fatalf("expected one signature, got %d", len(decoded.Value.Data))
	}
	if decoded.Value.Data[0] == nil {
		t.Fatal("decoded signature is nil")
	}

	return decoded.Value.Data[0]
}

func newTestAccount(t *testing.T) *core.Account {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	account, err := core.NewAccount(hexutil.Encode(crypto.FromECDSA(privateKey)))
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	return account
}

type unencodablePayload struct {
	Fn func()
}

func (u unencodablePayload) HashString() string {
	return "unencodable"
}
