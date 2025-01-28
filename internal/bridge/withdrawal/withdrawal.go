package withdrawal

import (
	"github.com/hyle-team/tss-svc/internal/db"
	"google.golang.org/protobuf/types/known/anypb"
)

type DepositSigningData interface {
	DepositIdentifier() db.DepositIdentifier
	ToPayload() *anypb.Any
}

type SigDataFormer[T DepositSigningData] interface {
	FormSigningData(deposit db.Deposit) (T, error)
}

type SigDataPayloader[T DepositSigningData] interface {
	FromPayload(payload *anypb.Any) (T, error)
}

type SigDataValidator[T DepositSigningData] interface {
	IsValid(data T, deposit db.Deposit) (bool, error)
}

type Constructor[T DepositSigningData] interface {
	SigDataFormer[T]
	SigDataValidator[T]
	SigDataPayloader[T]
}
