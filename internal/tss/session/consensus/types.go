package consensus

import (
	"crypto/sha256"
	"fmt"

	"github.com/hyle-team/tss-svc/internal/p2p"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	ToPayload() *anypb.Any
	p2p.Hashable
}

type Mechanism[T SigningData] interface {
	FormProposalData() (*T, error)

	FromPayload(payload *anypb.Any) (*T, error)
	VerifyProposedData(T) error
}
