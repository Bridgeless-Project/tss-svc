package consensus

import (
	"crypto/sha256"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p/broadcast"
	"google.golang.org/protobuf/proto"
)

type SignStartData struct {
	*p2p.SignStartData
}

func (s SignStartData) HashString() string {
	if s.SignStartData == nil {
		return ""
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.SignStartData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))
}

type SigningData interface {
	broadcast.Hashable
}

type Mechanism[T SigningData] interface {
	FormProposalData() (*T, error)
	VerifyProposedData(T) error
}
