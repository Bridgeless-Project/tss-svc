package test

import "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"

type Chain struct {
	Id            string `fig:"id,required"`
	Confirmations uint64 `fig:"confirmations,required"`
}

func FromChain(c chain.Chain) Chain {
	if c.Type != chain.TypeOther {
		panic("chain is not test")
	}

	return Chain{
		Id:            c.Id,
		Confirmations: c.Confirmations,
	}
}
