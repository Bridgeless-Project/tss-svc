package tss

import (
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v3/tss"
	"github.com/taurusgroup/multi-party-sig/pkg/party"
)

func ToECDSAPartyID(id core.Identifier) *tss.PartyID {
	return tss.NewPartyID(
		id.Name,
		id.Moniker,
		id.Pubkey,
	)
}

func ToFROSTPartyId(id core.Identifier) party.ID {
	return party.ID(fmt.Sprintf("%s_%s_%s", id.Name, id.Moniker, id.Pubkey.String()))
}
