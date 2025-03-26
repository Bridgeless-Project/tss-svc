package withdrawal

import (
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/tss/session/consensus"
	"google.golang.org/protobuf/types/known/anypb"
)

type DepositSigningData interface {
	consensus.SigningData
	DepositIdentifier() db.DepositIdentifier
}

type SigDataFormer[T DepositSigningData] interface {
	FormSigningData(deposit db.Deposit) (*T, error)
}

type SigDataPayloader[T DepositSigningData] interface {
	FromPayload(payload *anypb.Any) (*T, error)
}

type SigDataValidator[T DepositSigningData] interface {
	IsValid(data T, deposit db.Deposit) (bool, error)
}

type Constructor[T DepositSigningData] interface {
	SigDataFormer[T]
	SigDataValidator[T]
	SigDataPayloader[T]
}
