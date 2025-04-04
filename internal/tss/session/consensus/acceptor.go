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
				// there will be no data to sign in the current session
				if c.result.sigData == nil {
					c.logger.Info("got empty data to sign")
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
				return
			default:
				c.logger.Warn(fmt.Sprintf("unsupported request type %s from proposer", msg.Type))
			}
		}
	}
}

func (c *Consensus[T]) handleProposalMsg(msg consensusMsg) error {
	broadcastData := &p2p.ReliableBroadcastData{}
	if err := msg.Data.UnmarshalTo(broadcastData); err != nil {
		return errors.Wrap(err, "failed to unmarshal proposal data")
	}
	roundMsg, err := p2p.DecodeRoundMessage[T](broadcastData.GetRoundMsg())
	if err != nil {
		return errors.Wrap(err, "failed to decode round message")
	}

	valid := c.proposalBroadcaster.EnsureValid(p2p.ReliableBroadcastMsg[T]{
		Sender: msg.Sender,
		Msg:    roundMsg,
	})
	if !valid {
		return errors.New("proposal msg was not delivered reliable")
	}
	if roundMsg.Value == nil {
		return nil
	}

	proposalAccepted := false

	defer func() {
		dataRaw, _ := anypb.New(&p2p.AcceptanceData{Accepted: proposalAccepted})
		if err = c.broadcaster.Send(&p2p.SubmitRequest{
			Sender:    c.self.CosmosAddress().String(),
			SessionId: c.sessionId,
			Type:      p2p.RequestType_RT_ACCEPTANCE,
			Data:      dataRaw,
		}, c.proposer); err != nil {
			c.result.err = errors.Wrap(err, "failed to send proposal acceptance")
		}
	}()

	if err = c.mechanism.VerifyProposedData(*roundMsg.Value); err != nil {
		return errors.Wrap(err, "failed to verify proposed data")
	}

	c.result.sigData = roundMsg.Value
	proposalAccepted = true

	return nil
}

func (c *Consensus[T]) handleSignStartMsg(msg consensusMsg) error {
	broadcastData := &p2p.ReliableBroadcastData{}
	if err := msg.Data.UnmarshalTo(broadcastData); err != nil {
		return errors.Wrap(err, "failed to unmarshal proposal data")
	}
	roundMsg, err := p2p.DecodeRoundMessage[SignStartData](broadcastData.GetRoundMsg())
	if err != nil {
		return errors.Wrap(err, "failed to decode round message")
	}
	valid := c.signStartBroadcaster.EnsureValid(p2p.ReliableBroadcastMsg[SignStartData]{
		Sender: msg.Sender,
		Msg:    roundMsg,
	})
	if !valid {
		return errors.New("proposal msg was not delivered reliable")
	}

	if roundMsg.Value == nil {
		return errors.New("nil data in sign start message")
	}
	selectedParties := roundMsg.Value.GetParties()

	// validating if all selected parties are present and excluding local party
	signingParties := make([]p2p.Party, 0, len(selectedParties)-1)
	distinctParties := make(map[string]struct{}, len(selectedParties))
	selfPresent := false
	for _, participant := range selectedParties {
		if _, exists := distinctParties[participant]; exists {
			return errors.New(fmt.Sprintf("duplicate party '%s' in sign start message", participant))
		}
		distinctParties[participant] = struct{}{}

		if participant == c.self.CosmosAddress().String() {
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

	// local party does not participate in signing if not present in a sign start message
	if selfPresent {
		c.result.signers = signingParties
	}

	return nil
}
