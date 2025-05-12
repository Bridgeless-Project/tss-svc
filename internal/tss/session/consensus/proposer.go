package consensus

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
)

func (c *Consensus[T]) propose(ctx context.Context) {
	defer c.wg.Done()
	c.logger.Info("proposing data to sign...")

	signingData, err := c.mechanism.FormProposalData()
	if err != nil {
		c.result.err = errors.Wrap(err, "failed to form proposal data")
		return
	}

	broadcast := c.proposalBroadcaster.Broadcast(signingData)
	if !broadcast {
		c.result.err = errors.New("proposal data broadcast failure")
		return
	}

	// nothing to sign for this session
	if signingData == nil {
		c.logger.Info("no signing data were found")
		return
	}

	c.result.sigData = signingData
	c.logger.Info("data proposed, waiting for acceptances...")

	boundedCtx, cancel := context.WithTimeout(context.Background(), session.BoundaryProposalAcceptance)
	defer cancel()

	acceptances := Acceptances{}

	for {
		select {
		case <-ctx.Done():
			c.result.err = ctx.Err()
			return
		case <-boundedCtx.Done():
			c.logger.Info("collecting received acceptances...")

			possibleSigners := acceptances.Acceptors()
			// including proposer in total, possible signers count
			signersCount := len(possibleSigners) + 1
			// T+1 parties required for signing
			if signersCount <= c.threshold {
				c.result.err = errors.New("not enough parties accepted the proposal")
				return
			}

			// Selecting T signers (excluding proposer)
			signers := getSignersSet(possibleSigners, c.threshold, session.DeterministicRandSource(c.sessionId))
			c.result.signers = make([]p2p.Party, len(signers))
			for idx, party := range signers {
				c.result.signers[idx] = c.parties[party]
			}

			signStartMsg := &SignStartData{
				SignStartData: &p2p.SignStartData{
					Parties: append(signersToStr(signers), c.self.CosmosAddress().String()),
				},
			}
			broadcast = c.signStartBroadcaster.Broadcast(signStartMsg)
			if !broadcast {
				c.result.err = errors.New("sign start message broadcast failure")
				return
			}

			c.logger.Info("signing parties selected and notified")

			return
		case msg := <-c.msgs:
			if msg.Type != p2p.RequestType_RT_ACCEPTANCE {
				c.logger.Warn(fmt.Sprintf("unsupported proposalReq type %s from '%s'", msg.Type, msg.Sender))
				continue
			}
			if _, acceptanceExists := acceptances[msg.Sender]; acceptanceExists {
				c.logger.Warn(fmt.Sprintf("acceptance from '%s' already received, ignoring", msg.Sender))
				continue
			}

			result := &p2p.AcceptanceData{}
			if err = msg.Data.UnmarshalTo(result); err != nil {
				c.logger.Warn(fmt.Sprintf("failed to parse acceptance data from %s", msg.Sender))
				continue
			}

			acceptances[msg.Sender] = result.Accepted
			if !result.Accepted {
				c.logger.Warn(fmt.Sprintf("party '%s' NACK-ed the signing proposal", msg.Sender))
			}
		}
	}
}

func getSignersSet(signers []core.Address, threshold int, rand rand.Source) []core.Address {
	signersToRemove := len(signers) - threshold
	if signersToRemove <= 0 {
		return signers
	}

	for i := 0; i < signersToRemove; i++ {
		idx := rand.Uint64() % uint64(len(signers))
		signers = append(signers[:idx], signers[idx+1:]...)
	}

	return signers
}

type Acceptances map[core.Address]bool

func (a *Acceptances) Acceptors() []core.Address {
	var acceptors []core.Address
	for party, accepted := range *a {
		if accepted {
			acceptors = append(acceptors, party)
		}
	}
	return acceptors
}

func signersToStr(signers []core.Address) []string {
	var res []string
	for _, signer := range signers {
		res = append(res, signer.String())
	}
	return res
}
