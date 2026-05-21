package vault

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	vaultapi "github.com/hashicorp/vault/api"
)

func TestGetTssSharesLoadsSplitShares(t *testing.T) {
	ecdsaData := encodedECDSAShare(t)
	frostData := encodedFrostShare(t)
	storage := newTestStorage(t, map[string]map[string]interface{}{
		keyTssShareECDSA: ecdsaData,
		keyTssShareFROST: frostData,
	})

	shares, err := storage.GetTssShares()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shares.Share == nil {
		t.Fatal("expected ecdsa share")
	}
	if shares.FrostShare == nil {
		t.Fatal("expected frost share")
	}
}

func TestGetTssSharesLoadsLegacyECDSAFallback(t *testing.T) {
	storage := newTestStorage(t, map[string]map[string]interface{}{
		legacyKeyTssShare: encodedECDSAShare(t),
	})

	shares, err := storage.GetTssShares()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shares.Share == nil {
		t.Fatal("expected legacy ecdsa share")
	}
}

func TestGetTssSharesPrefersNewECDSAOverLegacy(t *testing.T) {
	newShare := encodedECDSAShareWithThreshold(t, 2)
	legacyShare := encodedECDSAShareWithThreshold(t, 1)
	storage := newTestStorage(t, map[string]map[string]interface{}{
		keyTssShareECDSA:  newShare,
		legacyKeyTssShare: legacyShare,
	})

	shares, err := storage.GetTssShares()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shares.Share == nil {
		t.Fatal("expected ecdsa share")
	}
	if len(shares.Share.Ks) != 1 || shares.Share.Ks[0].Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("expected new share to take precedence")
	}
}

func TestSaveTssShareUsesProvidedKey(t *testing.T) {
	store := map[string]map[string]interface{}{}
	storage := newTestStorage(t, store)

	if err := storage.SaveTssShare(secrets.TssShareKeyTemporary, &keygen.LocalPartySaveData{Ks: []*big.Int{big.NewInt(7)}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := store[keyTssShareTemp]; !ok {
		t.Fatalf("expected share saved under temporary key")
	}
	if _, ok := store[keyTssShareECDSA]; ok {
		t.Fatalf("did not expect temporary share saved under primary ecdsa key")
	}
}

func TestGetTemporaryTssShare(t *testing.T) {
	storage := newTestStorage(t, map[string]map[string]interface{}{
		keyTssShareTemp: encodedECDSAShareWithThreshold(t, 3),
	})

	share, err := storage.GetTemporaryTssShare()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if share == nil {
		t.Fatal("expected temporary share")
	}
	if len(share.Ks) != 1 || share.Ks[0].Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("unexpected temporary share")
	}
}

func newTestStorage(t *testing.T, data map[string]map[string]interface{}) *Storage {
	t.Helper()
	return &Storage{client: testKVStore{data: data}}
}

func encodedECDSAShare(t *testing.T) map[string]interface{} {
	t.Helper()
	return encodedECDSAShareWithThreshold(t, 0)
}

func encodedECDSAShareWithThreshold(t *testing.T, threshold int) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(&keygen.LocalPartySaveData{Ks: []*big.Int{big.NewInt(int64(threshold))}})
	if err != nil {
		t.Fatalf("failed to marshal ecdsa share: %v", err)
	}

	return map[string]interface{}{
		valueKey: string(raw),
	}
}

func encodedFrostShare(t *testing.T) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		protocolKey: protocolFrost,
		encodingKey: encodingCBOR,
		valueKey:    testFrostShare,
	}
}

const testFrostShare = "pmJJRHgtYnJpZGdlMWRrdHUwM2p5dXhkbnh4dHgyazcycnd2OHh5dXo1cXF4YTd1azZlaVRocmVzaG9sZAFsUHJpdmF0ZVNoYXJlWCC9N/9lka71zG3jZl8soWXc/uceDkNbh8xV88Wg8fnu9GlQdWJsaWNLZXlYIQJIOAfR6XMiLu6n22Ud3I+IuBrudJ7oyYfOimZXIe0GXWhDaGFpbktlefZyVmVyaWZpY2F0aW9uU2hhcmVzWPejeC1icmlkZ2Uxc2Nsa3NyOHZudmx6aDRoOTc3OHV2YXI1M2U5cjI1ZGZqMnhrZXlYIQJf/ddneZq9RHqLvZR8QaAmRbmYvN9skrJPmJJlfX0JN3gtYnJpZGdlMTVwNmN4ZDNhOTJydGNreTBsOHdmNmhsOTk5eXM1dm10ZzZ4MmFzWCEDjHMFEXxz1/nEFhklKzx40ciZaH7ASbRi08xEDxUE92Z4LWJyaWRnZTFka3R1MDNqeXV4ZG54eHR4Mms3MnJ3djh4eXV6NXFxeGE3dWs2ZVghAjD93D5Pe43c92+2btNoSDTxasbdHTei9etQc43a8s4c"

type testKVStore struct {
	data map[string]map[string]interface{}
}

func (s testKVStore) Get(_ context.Context, secretPath string) (*vaultapi.KVSecret, error) {
	value, ok := s.data[secretPath]
	if !ok {
		return nil, vaultapi.ErrSecretNotFound
	}

	return &vaultapi.KVSecret{Data: value}, nil
}

func (s testKVStore) Put(_ context.Context, secretPath string, data map[string]interface{}, _ ...vaultapi.KVOption) (*vaultapi.KVSecret, error) {
	if s.data == nil {
		s.data = make(map[string]map[string]interface{})
	}
	s.data[secretPath] = data

	return &vaultapi.KVSecret{}, nil
}
