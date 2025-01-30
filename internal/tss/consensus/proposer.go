package consensus

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

var pendingWithdrawalStatus = types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING

func (c *Consensus[T]) propose(ctx context.Context) {
	defer c.wg.Done()

	depositToSign, err := c.db.GetWithSelector(db.DepositsSelector{
		ChainId: &c.chainId,
		Status:  &pendingWithdrawalStatus,
		One:     true,
	})
	if err != nil {
		c.result.err = errors.Wrap(err, "failed to get deposit to sign")
		return
	}

	proposalReq := &p2p.SubmitRequest{
		Sender:    c.self.String(),
		SessionId: c.sessionId,
		Type:      p2p.RequestType_RT_PROPOSAL,
		Data:      nil,
	}

	if depositToSign != nil {
		signData, err := c.constructor.FormSigningData(*depositToSign)
		if err != nil {
			c.result.err = errors.Wrap(err, "failed to form signing data")
			return
		}

		c.result.sigData = &signData
		proposalReq.Data = signData.ToPayload()
	}

	c.broadcaster.Broadcast(proposalReq)

	// nothing to sign for this session
	if depositToSign == nil {
		return
	}

	boundedCtx, cancel := context.WithTimeout(context.Background(), tss.BoundaryAcceptance)
	defer cancel()

	acceptances := Acceptances{}

	for {
		select {
		case <-ctx.Done():
			c.result.err = ctx.Err()
			return
		case <-boundedCtx.Done():
			possibleSigners := acceptances.Acceptors()
			// including proposer in total possible signers count
			signersCount := len(possibleSigners) + 1
			// T+1 parties required for signing
			if signersCount <= c.threshold {
				c.result.err = errors.New("not enough parties accepted the proposal")
				return
			}

			// Selecting T signers (excluding proposer)
			signers := getSignersSet(possibleSigners, c.threshold, deterministicRandSource(c.sessionId))
			c.result.signers = make([]p2p.Party, len(signers))
			for idx, party := range signers {
				c.result.signers[idx] = c.parties[party]
			}

			signStartData, _ := anypb.New(&p2p.SignStartData{Parties: append(signersToStr(signers), c.self.String())})
			msg := &p2p.SubmitRequest{
				Sender:    c.self.String(),
				SessionId: c.sessionId,
				Type:      p2p.RequestType_RT_SIGN_START,
				Data:      signStartData,
			}
			p2p.NewBroadcaster(c.result.signers).Broadcast(msg)

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
	signersToRemove := threshold - len(signers)
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
