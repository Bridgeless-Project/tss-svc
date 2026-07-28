package secrets

import (
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/pkg/errors"
)

func MarshalTssShareSet(results []tss.Share) (map[TssShareKey][]byte, error) {
	if len(results) == 0 {
		return nil, errors.New("no TSS shares to save")
	}
	shares := make(map[TssShareKey][]byte, len(results))
	protocols := make(map[tss.ProtocolType]struct{}, len(results))
	for _, result := range results {
		if result == nil {
			return nil, errors.New("nil TSS share")
		}
		if _, exists := protocols[result.Protocol()]; exists {
			return nil, errors.Errorf("duplicate TSS share protocol %s", result.Protocol())
		}
		raw, err := result.Marshal()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal %s TSS share", result.Protocol())
		}
		protocols[result.Protocol()] = struct{}{}
		shares[TssShareKey(result.GetVaultPath())] = raw
	}
	return shares, nil
}

// SaveTssShareSet serializes the complete set before publishing anything. A
// batch-capable store activates all shares in one backing-store update.
func SaveTssShareSet(storage Storage, results []tss.Share) error {
	shares, err := MarshalTssShareSet(results)
	if err != nil {
		return err
	}
	if batchStorage, ok := storage.(BatchTssShareStorage); ok {
		return batchStorage.SaveTssShares(shares)
	}
	for key, raw := range shares {
		if err = storage.SaveTssShare(key, raw); err != nil {
			return errors.Wrapf(err, "failed to save TSS share %s", key)
		}
	}
	return nil
}
