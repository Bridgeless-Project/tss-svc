package tss

import "time"

const (
	BoundaryKeygenSession  = time.Minute
	BoundarySigningSession = BoundarySign + BoundaryConsensus
	BoundarySign           = 10 * time.Second
	BoundaryConsensus      = BoundaryAcceptance + 5*time.Second
	BoundaryAcceptance     = 5 * time.Second
)
