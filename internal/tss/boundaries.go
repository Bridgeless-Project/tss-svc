package tss

import "time"

const (
	BoundaryKeygenSession  = time.Minute
	BoundarySigningSession = 10 * time.Second

	BoundaryAcceptance = 5 * time.Second
	BoundaryConsensus  = BoundaryAcceptance + 5*time.Second
)
