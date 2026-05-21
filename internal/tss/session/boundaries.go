package session

import "time"

const (
	BoundaryKeygenSession = time.Minute

	BoundarySigningSession = BoundaryConsensus + BoundarySign + BoundarySignatureDistribution + BoundaryFinalize

	BoundaryConsensus             = BoundaryProposalAcceptance + 10*time.Second
	BoundaryProposalAcceptance    = 5 * time.Second
	BoundarySign                  = 13 * time.Second
	BoundarySignatureDistribution = 5 * time.Second
	BoundaryFinalize              = 7 * time.Second

	BoundaryBitcoinSignRoundDelay = 500 * time.Millisecond
)
