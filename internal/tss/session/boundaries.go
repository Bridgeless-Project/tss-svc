package session

import "time"

const (
	BoundaryKeygenSession  = time.Minute
	BoundarySigningSession = BoundarySign + BoundaryConsensus + BoundaryFinalize
	BoundarySign           = 10 * time.Second
	BoundaryConsensus      = BoundaryAcceptance + 5*time.Second
	BoundaryAcceptance     = 5 * time.Second
	BoundaryFinalize       = 10 * time.Second

	BoundaryBitcoinSingRoundDelay = 500 * time.Millisecond
)
