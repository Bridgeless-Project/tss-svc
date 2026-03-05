package withdrawal

import (
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
)

type DepositSigningData interface {
	consensus.SigningData
	DepositIdentifiers() []db.DepositIdentifier //Shared interface returns a slice to support processing more than one deposit on evm
} //Other chains support processing of only one deposit at a time

type SigDataFormer[T DepositSigningData] interface {
	FormSigningData(deposits ...db.Deposit) (*T, error)
}

type SigDataValidator[T DepositSigningData] interface {
	IsValid(data T, deposits ...db.Deposit) (bool, error)
}

type Constructor[T DepositSigningData] interface {
	SigDataFormer[T]
	SigDataValidator[T]
}
