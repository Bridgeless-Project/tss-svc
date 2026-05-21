package run

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
)

type keygenTestStorage struct{}

func TestStoreKeygenResultFileUsesOwnerOnlyPermissions(t *testing.T) {
	prevOutputType := utils.OutputType
	prevFilePath := utils.FilePath
	t.Cleanup(func() {
		utils.OutputType = prevOutputType
		utils.FilePath = prevFilePath
	})

	utils.OutputType = "file"
	utils.FilePath = filepath.Join(t.TempDir(), "share.json")

	if err := storeKeygenResult(&keygen.LocalPartySaveData{}, keygenTestStorage{}, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(utils.FilePath)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0644 {
		t.Fatalf("expected file permissions 0644, got %o", got)
	}
}

func TestStoreKeygenResultRejectsUnknownOutput(t *testing.T) {
	prevOutputType := utils.OutputType
	t.Cleanup(func() { utils.OutputType = prevOutputType })

	utils.OutputType = "unknown"
	if err := storeKeygenResult(&keygen.LocalPartySaveData{}, keygenTestStorage{}, 0); err == nil {
		t.Fatal("expected error")
	}
}

func (keygenTestStorage) GetKeygenPreParams() (*keygen.LocalPreParams, error) {
	return nil, nil
}

func (keygenTestStorage) SaveKeygenPreParams(*keygen.LocalPreParams) error {
	return nil
}

func (keygenTestStorage) GetCoreAccount() (*core.Account, error) {
	return nil, nil
}

func (keygenTestStorage) SaveCoreAccount(*core.Account) error {
	return nil
}

func (keygenTestStorage) SaveTssShare(interface{}) error {
	return nil
}

func (keygenTestStorage) GetTssShare() (interface{}, int, error) {
	return nil, 0, nil
}

func (keygenTestStorage) GetTssShares() (*secrets.TssShares, error) {
	return nil, nil
}

func (keygenTestStorage) SaveLocalPartyTlsCertificate(_, _ []byte) error {
	return nil
}

func (keygenTestStorage) GetLocalPartyTlsCertificate() (*tls.Certificate, error) {
	return nil, nil
}
