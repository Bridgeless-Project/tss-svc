package consensus

import (
	"google.golang.org/protobuf/types/known/anypb"
)

type SigningData interface {
	ToPayload() *anypb.Any
}

type Mechanism[T SigningData] interface {
	FormProposalData() (*T, error)

	FromPayload(payload *anypb.Any) (*T, error)
	VerifyProposedData(T) error
}
