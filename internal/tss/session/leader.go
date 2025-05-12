package session

import (
	"crypto/sha256"
	"math/rand/v2"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
)

// TODO: add more randomness to the seed
func DeterministicRandSource(sessionId string) rand.Source {
	seed := sha256.Sum256([]byte(sessionId))
	return rand.NewChaCha8(seed)
}

func DetermineLeader(sessionId string, partyIds tss.SortedPartyIDs) core.Address {
	generator := DeterministicRandSource(sessionId)
	proposerIdx := int(generator.Uint64() % uint64(partyIds.Len()))

	return core.AddrFromPartyId(partyIds[proposerIdx])
}

func SortAllParties(parties []p2p.Party, self core.Address) tss.SortedPartyIDs {
	totalPartiesCount := len(parties) + 1

	partyIds := make([]*tss.PartyID, totalPartiesCount)
	for idx, party := range parties {
		partyIds[idx] = party.Identifier()
	}
	partyIds[totalPartiesCount-1] = self.PartyIdentifier()

	return tss.SortPartyIDs(partyIds)
}
