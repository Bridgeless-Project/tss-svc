package p2p

import (
	"math/big"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v3/tss"
	"google.golang.org/grpc"
)

const DefaultConnectionTimeout = time.Second

type Party struct {
	CoreAddress core.Address

	connection *grpc.ClientConn
	pemCert    []byte
	identifier *tss.PartyID
}

func (p *Party) Identifier() *tss.PartyID {
	return p.identifier
}

func (p *Party) Connection() *grpc.ClientConn {
	return p.connection
}

func (p *Party) Key() *big.Int {
	return p.CoreAddress.PartyKey()
}

func (p *Party) PEMCert() []byte {
	return p.pemCert
}

func NewParty(coreAddr core.Address, connection *grpc.ClientConn, pemCert []byte) Party {
	return Party{
		pemCert:     pemCert,
		connection:  connection,
		CoreAddress: coreAddr,
		identifier:  coreAddr.PartyIdentifier(),
	}
}

// MergeParties takes slice of parties and merges them into a single slice without duplicates based on their CoreAddress.
// The order of the parties is preserved, with the first occurrence of each unique CoreAddress being retained in the merged result.
func MergeParties(parties ...Party) []Party {
	var (
		merged  []Party
		present = make(map[string]struct{})
	)

	for _, party := range parties {
		key := party.CoreAddress.String()
		if _, ok := present[key]; ok {
			continue
		}

		present[key] = struct{}{}
		merged = append(merged, party)
	}

	return merged
}
