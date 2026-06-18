package core

import "time"

type EventDataCommissionCollection struct {
	EpochId     uint32
	BlockHeight int64
}

type EventCommissionCollection struct {
	EventDataCommissionCollection
	Time time.Time
}
