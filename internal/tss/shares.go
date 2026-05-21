package tss

import (
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/pkg/errors"
	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
)

func ECDSAShareFromProtocol(share interface{}, protocolID int) (*keygen.LocalPartySaveData, error) {
	if protocolID != ProtocolID_ECDSA {
		return nil, errors.New("resharing supports only ECDSA TSS shares")
	}

	return ECDSAShare(share)
}

func ECDSAShare(share interface{}) (*keygen.LocalPartySaveData, error) {
	switch typedShare := share.(type) {
	case *keygen.LocalPartySaveData:
		if typedShare == nil {
			return nil, errors.New("nil ECDSA TSS share")
		}
		return typedShare, nil
	case keygen.LocalPartySaveData:
		return &typedShare, nil
	default:
		return nil, errors.Errorf("expected ECDSA TSS share, got %T", share)
	}
}

func FrostShare(share interface{}) (*frostkeygen.Config, error) {
	switch typedShare := share.(type) {
	case *frostkeygen.Config:
		if typedShare == nil {
			return nil, errors.New("nil FROST TSS share")
		}
		return typedShare, nil
	case frostkeygen.Config:
		return &typedShare, nil
	default:
		return nil, errors.Errorf("expected FROST TSS share, got %T", share)
	}
}
