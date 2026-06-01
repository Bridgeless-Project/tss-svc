package tss

import (
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v3/tss"
)

func ToECDSAPartyID(id core.Identifier) *tss.PartyID {
	return tss.NewPartyID(
		id.Name,
		id.Moniker,
		id.Pubkey,
	)
}

//func ToFROSTPartyId(id core.Identifier) party.ID {
//	return party.ID(fmt.Sprintf("%s_%s_%s", id.Name, id.Moniker, id.Pubkey.String()))
//}
