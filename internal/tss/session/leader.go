package session

import (
	"crypto/sha256"
	"math/rand/v2"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	tsslib "github.com/bnb-chain/tss-lib/v3/tss"
)

// TODO: add more randomness to the seed
func DeterministicRandSource(sessionId string) rand.Source {
	seed := sha256.Sum256([]byte(sessionId))
	return rand.NewChaCha8(seed)
}

func DetermineLeader(sessionId string, partyIds tsslib.SortedPartyIDs) core.Address {
	generator := DeterministicRandSource(sessionId)
	proposerIdx := int(generator.Uint64() % uint64(partyIds.Len()))

	return core.AddrFromString(partyIds[proposerIdx].Moniker)
}

func SortAllParties(parties []p2p.Party, self core.Address) tsslib.SortedPartyIDs {
	totalPartiesCount := len(parties) + 1

	partyIds := make([]*tsslib.PartyID, totalPartiesCount)
	for idx, party := range parties {
		partyIds[idx] = tss.ToECDSAPartyID(party.Identifier())
	}
	partyIds[totalPartiesCount-1] = tss.ToECDSAPartyID(self.PartyIdentifier())

	return tsslib.SortPartyIDs(partyIds)
}
