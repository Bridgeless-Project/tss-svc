package consensus

import (
	"context"
	"fmt"

	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
)

func (c *Consensus[T]) accept(ctx context.Context) {
	defer c.wg.Done()
	c.logger.Info("accepting proposal...")

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
					c.logger.Info("got empty data to sign")
					c.logger.Info("consensus finished")
					return
				}

				proposalAccepted = true
				c.logger.Info("proposal accepted, waiting for sign start message...")
			case p2p.RequestType_RT_SIGN_START:
				if !proposalAccepted {
					c.logger.Warn("sign start message received before proposal, ignoring")
					continue
				}

				if err := c.handleSignStartMsg(msg); err != nil {
					c.result.err = errors.Wrap(err, "failed to handle sign start message")
				}

				c.logger.Info("sign start message with signing parties received")
				c.logger.Info("consensus finished")
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

	data, err := c.constructor.FromPayload(msg.Data)
	if err != nil {
		return errors.Wrap(err, "failed to load consensus payload")
	}

	deposit, err := c.db.Get(data.DepositIdentifier())
	if err != nil {
		return errors.Wrap(err, "failed to get deposit data")
	}

	if deposit == nil {
		deposit, err = c.processor.FetchDeposit(data.DepositIdentifier())
		if err != nil {
			// TODO: err check?
			return errors.Wrap(err, "failed to fetch deposit data")
		}
		if _, err = c.db.Insert(*deposit); err != nil {
			return errors.Wrap(err, "failed to save deposit data")
		}
	}

	if deposit.WithdrawalStatus != types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING {
		return errors.New("deposit is not in pending status")
	}

	isValid, err := c.constructor.IsValid(data, *deposit)
	if err != nil {
		return errors.Wrap(err, "failed to validate signing data")
	}
	if !isValid {
		return errors.New("invalid signing data")
	}

	c.result.sigData = &data
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

	// local party does not participate in signing if not present in sign start message
	if selfPresent {
		c.result.signers = signingParties
	}

	return nil
}
