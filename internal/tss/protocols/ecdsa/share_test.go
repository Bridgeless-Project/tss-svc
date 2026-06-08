package tss

import (
	"testing"

	"github.com/bnb-chain/tss-lib/v3/common"
	tsscrypto "github.com/bnb-chain/tss-lib/v3/crypto"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/proto"
)

func TestEcdsaShareVerify(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	hash := crypto.Keccak256([]byte("message"))
	rawSignature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		t.Fatalf("sign hash: %v", err)
	}

	signature, err := proto.Marshal(&common.SignatureData{
		Signature:         rawSignature[:64],
		SignatureRecovery: rawSignature[64:],
		R:                 rawSignature[:32],
		S:                 rawSignature[32:64],
		M:                 hash,
	})
	if err != nil {
		t.Fatalf("marshal signature: %v", err)
	}

	share := NewEcdsaShare()
	share.data = &keygen.LocalPartySaveData{
		ECDSAPub: tsscrypto.NewECPointNoCurveCheck(
			privateKey.Curve,
			privateKey.PublicKey.X,
			privateKey.PublicKey.Y,
		),
	}

	valid, err := share.Verify(signature, hash)
	if err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	if !valid {
		t.Fatal("expected signature to be valid")
	}

	valid, err = share.Verify(signature, crypto.Keccak256([]byte("other message")))
	if err != nil {
		t.Fatalf("verify altered hash: %v", err)
	}
	if valid {
		t.Fatal("expected signature for altered hash to be invalid")
	}
}

func TestEcdsaShareVerifyRejectsMalformedSignature(t *testing.T) {
	share := NewEcdsaShare()
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	share.data = &keygen.LocalPartySaveData{
		ECDSAPub: tsscrypto.NewECPointNoCurveCheck(
			privateKey.Curve,
			privateKey.PublicKey.X,
			privateKey.PublicKey.Y,
		),
	}

	if valid, err := share.Verify([]byte("not protobuf"), make([]byte, 32)); err == nil || valid {
		t.Fatal("expected malformed signature to be rejected")
	}

	signature, err := proto.Marshal(&common.SignatureData{R: []byte{1}})
	if err != nil {
		t.Fatalf("marshal incomplete signature: %v", err)
	}
	if valid, err := share.Verify(signature, make([]byte, 32)); err == nil || valid {
		t.Fatal("expected signature without S to be rejected")
	}
}
