package p2p

import (
	"math/big"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/bnb-chain/tss-lib/v2/tss"
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

// MergeParties takes multiple slices of parties and merges them
// into a single slice without duplicates based on the CoreAddress.
func MergeParties(parties ...Party) []Party {
	var (
		merged  []Party
		present = make(map[string]struct{})
	)

	for _, party := range parties {
		if _, ok := present[party.CoreAddress.String()]; !ok {
			merged = append(merged, party)
		}
	}

	return merged
}
