package commissions

import "github.com/Bridgeless-Project/tss-svc/internal/core"

// TODO:
// - get start time from blocktime
// - get all tokens on block x
// - get commission by each tokenId that present on Bridgeless (?)
// - form N signing datas for each token
// - consensus just to select signing parties from the whole set

// session:
// - consensus
// - signing
// - distrib
// - finalize
type Session struct {
	data core.EventDataCommissionCollection
}
