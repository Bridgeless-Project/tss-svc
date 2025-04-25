package tss

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/hyle-team/tss-svc/internal/core"
)

const (
	OutChannelSize = 1000
	EndChannelSize = 1
	MsgsCapacity   = 100
)

type partyMsg struct {
	Sender      core.Address
	WireMsg     []byte
	IsBroadcast bool
}

func MaxMaliciousParties(partiesCount, threshold int) int {
	// T+1 parties are required to function
	return partiesCount - (threshold + 1)
}

type Signatures struct {
	Data []*common.SignatureData
}

func (s Signatures) HashString() string {
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	_ = encoder.Encode(s)

	return fmt.Sprintf("%x", sha256.Sum256(buff.Bytes()))
}
