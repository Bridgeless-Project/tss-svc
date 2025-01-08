package p2p

import (
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"google.golang.org/grpc"
)

type Account string

func (a Account) String() string {
	return string(a)
}

type Party struct {
	PubKey      string
	CoreAddress core.Address

	connection *grpc.ClientConn
	identifier *tss.PartyID
}

func (p *Party) Identifier() *tss.PartyID {
	return p.identifier
}

func (p *Party) Connection() *grpc.ClientConn {
	return p.connection
}

func NewParty(pubKey string, coreAddr core.Address, connection *grpc.ClientConn) Party {
	return Party{
		PubKey:      pubKey,
		connection:  connection,
		CoreAddress: coreAddr,
		identifier:  AddrToPartyIdentifier(coreAddr),
	}
}

func AddrToPartyIdentifier(addr core.Address) *tss.PartyID {
	return tss.NewPartyID(
		addr.String(),
		addr.String(),
		new(big.Int).SetBytes(addr.Bytes()),
	)
}

func AddrFromPartyIdentifier(id *tss.PartyID) (core.Address, error) {
	return core.AddressFromString(id.GetMoniker())
}
