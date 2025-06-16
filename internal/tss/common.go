package tss

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v2/common"
	"google.golang.org/protobuf/proto"
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
	if len(s.Data) == 0 {
		return ""
	}

	var buff bytes.Buffer

	for _, sig := range s.Data {
		if sig == nil {
			continue
		}

		data, _ := proto.MarshalOptions{Deterministic: true}.Marshal(sig)
		buff.Write(data)
	}

	return fmt.Sprintf("%x", sha256.Sum256(buff.Bytes()))
}
