package session

import "time"

const (
	BoundaryKeygenSession = time.Minute

	BoundarySigningSession = BoundaryConsensus + BoundarySign + BoundarySignatureDistribution + BoundaryFinalize

	BoundaryConsensus             = BoundaryProposalAcceptance + 15*time.Second
	BoundaryProposalAcceptance    = 5 * time.Second
	BoundarySign                  = 20 * time.Second
	BoundarySignatureDistribution = 5 * time.Second
	BoundaryFinalize              = 15 * time.Second

	BoundaryBitcoinSignRoundDelay = 500 * time.Millisecond
)
