package sharefile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/pkg/errors"
)

func ProtocolPath(base string, protocol tss.ProtocolType) string {
	ext := filepath.Ext(base)
	if ext == "" {
		return fmt.Sprintf("%s.%s", base, protocol)
	}
	return fmt.Sprintf("%s.%s%s", base[:len(base)-len(ext)], protocol, ext)
}

// WriteSet writes every protocol share with owner-only permissions. Existing
// files are never overwritten, and files created by a failed set are removed.
func WriteSet(base string, shares map[tss.ProtocolType][]byte) ([]string, error) {
	if len(shares) == 0 {
		return nil, errors.New("no key shares to write")
	}
	protocols := make([]tss.ProtocolType, 0, len(shares))
	for protocol := range shares {
		protocols = append(protocols, protocol)
	}
	sort.Slice(protocols, func(i, j int) bool { return protocols[i] < protocols[j] })

	paths := make(map[tss.ProtocolType]string, len(protocols))
	for _, protocol := range protocols {
		paths[protocol] = ProtocolPath(base, protocol)
		if _, err := os.Stat(paths[protocol]); err == nil {
			return nil, errors.Errorf("refusing to overwrite existing key share file %s", paths[protocol])
		} else if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to inspect key share output path")
		}
	}

	created := make([]string, 0, len(protocols))
	for _, protocol := range protocols {
		path := paths[protocol]
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			_, err = file.Write(shares[protocol])
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
		}
		if err != nil {
			for _, createdPath := range created {
				_ = os.Remove(createdPath)
			}
			return nil, errors.Wrapf(err, "failed to write %s keygen result", protocol)
		}
		created = append(created, path)
	}
	return created, nil
}
