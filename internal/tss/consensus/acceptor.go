package consensus

import (
	"context"
	"fmt"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

func (c *Consensus[T]) accept(ctx context.Context) {
	defer c.wg.Done()

	var proposalAccepted bool

	for {
		select {
		case <-ctx.Done():
			c.result.err = ctx.Err()
			return
		case msg := <-c.msgs:
			if msg.Sender != c.proposer {
				c.logger.Warn(fmt.Sprintf("message sender %s is not proposer", msg.Sender))
				continue
			}
			switch msg.Type {
			case p2p.RequestType_RT_PROPOSAL:
				if proposalAccepted {
					c.logger.Warn("proposal message received after proposal accepted, ignoring")
					continue
				}

				if err := c.handleProposalMsg(msg); err != nil {
					c.result.err = errors.Wrap(err, "failed to handle proposal message")
					return
				}
				// there will be no data to sign in current session
				if c.result.sigData == nil {
					return
				}

				proposalAccepted = true
			case p2p.RequestType_RT_SIGN_START:
				if !proposalAccepted {
					c.logger.Warn("sign start message received before proposal, ignoring")
					continue
				}

				if err := c.handleSignStartMsg(msg); err != nil {
					c.result.err = errors.Wrap(err, "failed to handle sign start message")
				}

				return
			default:
				c.logger.Warn(fmt.Sprintf("unsupported request type %s from proposer", msg.Type))
			}
		}
	}
}

func (c *Consensus[T]) handleProposalMsg(msg consensusMsg) error {
	if msg.Data == nil {
		// there is no data to sign in current session
		return nil
	}

	proposalAccepted := false

	defer func() {
		// TODO: do not depend on the error
		dataRaw, _ := anypb.New(&p2p.AcceptanceData{Accepted: proposalAccepted})
		if err := c.broadcaster.Send(&p2p.SubmitRequest{
			Sender:    c.self.String(),
			SessionId: c.sessionId,
			Type:      p2p.RequestType_RT_ACCEPTANCE,
			Data:      dataRaw,
		}, c.proposer); err != nil {
			c.result.err = errors.Wrap(err, "failed to send proposal acceptance")
		}
	}()

	var data T
	if err := data.FromPayload(msg.Data); err != nil {
		return errors.Wrap(err, "failed to load consensus payload")
	}

	_ = data.Deposit()
	// TODO: check is deposit is in db and not processed yet

	existsInDb := true
	if !existsInDb {
		// TODO: get deposit from chain + core info about token
	}

	isValid, err := c.sigDataValidator.IsValid(data)
	if err != nil {
		return errors.Wrap(err, "failed to validate signing data")
	}
	if !isValid {
		return errors.New("invalid signing data")
	}

	c.result.sigData = data
	proposalAccepted = true

	return nil
}

func (c *Consensus[T]) handleSignStartMsg(msg consensusMsg) error {
	if msg.Data == nil {
		return errors.New("nil data in sign start message")
	}

	signStartData := &p2p.SignStartData{}
	if err := msg.Data.UnmarshalTo(signStartData); err != nil {
		return errors.Wrap(err, "failed to unmarshal sign start data")
	}

	// validating if all parties are present and excluding local party
	signingParties := make([]p2p.Party, 0, len(signStartData.Parties)-1)
	selfPresent := false
	for _, participant := range signStartData.Parties {
		if participant == c.self.String() {
			selfPresent = true
			continue
		}

		addr, err := core.AddressFromString(participant)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to parse party address '%s'", participant))
		}

		party, exists := c.parties[addr]
		if !exists {
			return errors.New(fmt.Sprintf("party '%s' is not present in consensus", addr.String()))
		}

		signingParties = append(signingParties, party)
	}

	// local party does not participate in signing
	if selfPresent {
		c.result.signers = signingParties
	}

	return nil
}
