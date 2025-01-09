package p2p

import (
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
	"google.golang.org/grpc"
)

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

func (p *Party) Key() *big.Int {
	return p.CoreAddress.PartyKey()
}

func NewParty(pubKey string, coreAddr core.Address, connection *grpc.ClientConn) Party {
	return Party{
		PubKey:      pubKey,
		connection:  connection,
		CoreAddress: coreAddr,
		identifier:  coreAddr.PartyIdentifier(),
	}
}
