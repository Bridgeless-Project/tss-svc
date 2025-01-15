package consensus

import (
	"bytes"
	"encoding/json"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
)

type partyMsg struct {
	Type        p2p.RequestType
	Sender      core.Address
	WireMsg     []byte
	IsBroadcast bool
}

// constructs c.resultSigners basing on slice of partie`s public keys came from proposer
func (c *Consensus) receiveSignerNotification(msg partyMsg) {
	var signersList []string
	err := json.Unmarshal(msg.WireMsg, &signersList)
	if err != nil {
		c.logger.Error("failed to unmarshal signer list", err)
		c.err = errors.Wrap(err, "failed to unmarshal signer list")
		return
	}
	for _, signer := range signersList {
		for _, party := range c.parties {
			if party.CoreAddress.PartyKey().String() == signer {
				c.resultSigners = append(c.resultSigners, party)
			}
		}
	}
	c.logger.Info("Signer list received", c.resultSigners)
}

func (c *Consensus) validateIncomingData(msg partyMsg) error {
	//perform validation by signer
	//validate sender
	senderId := msg.Sender.PartyIdentifier().Id
	if senderId != c.proposerKey {
		c.logger.Error("invalid proposer")
		c.err = errors.New("invalid proposer")
		return c.err
	}
	// validate deposit data with recreating it
	localData, err := c.formData(c.data)
	if err != nil {
		c.logger.Error("failed to form data", err)
		c.err = err
		return c.err
		c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
	}
	if !bytes.Equal(localData, msg.WireMsg) {
		c.logger.Error("invalid data")
		c.err = errors.Wrap(errors.New("invalid data"), "formed different data")
		return c.err
		c.sendMessage(nil, msg.Sender.PartyIdentifier(), p2p.RequestType_NACK)
	}
	c.logger.Info("got new data: ", msg.WireMsg)
	c.resultData = msg.WireMsg
	return nil
}

func (c *Consensus) notifySigners(signers []string) error {
	resultSignersData, err := json.Marshal(signers)
	if err != nil {
		return errors.Wrap(err, "failed to serialize resultSigners")
	}

	for _, signer := range c.resultSigners {
		c.sendMessage(resultSignersData, signer.CoreAddress.PartyIdentifier(), p2p.RequestType_SIGNER_NOTIFY)
	}
	return nil
}

func (c *Consensus) handleACK(msg partyMsg, votesCount *int) {
	if _, exists := c.ackSet[msg.Sender.PartyKey().String()]; !exists {
		c.ackSet[msg.Sender.PartyKey().String()] = struct{}{}
		*votesCount++
		c.logger.Info("number voted: ")
		c.logger.Info("Received ACK from party", msg.Sender.PartyIdentifier())
	}
}

func (c *Consensus) finalizeConsensus() {
	c.logger.Info("All parties voted")
	if len(c.ackSet) < c.threshold {
		c.logger.Error("Didn`t reach threshold")
		c.err = errors.Wrap(errors.New("consensus failed"), "didn`t reach threshold")
	}
	var signerKeysList []string
	for signerKey, _ := range c.ackSet {
		for _, party := range c.parties {
			if party.CoreAddress.PartyKey().String() == signerKey {
				c.resultSigners = append(c.resultSigners, party)
				break
			}
		}
		signerKeysList = append(signerKeysList, signerKey)
	}
	c.resultSigners = c.resultSigners[:c.threshold]
	c.logger.Info("Signers list", c.resultSigners)

	err := c.notifySigners(signerKeysList)
	if err != nil {
		c.logger.Error("failed to notify signers", err)
		c.err = errors.Wrap(err, "failed to notify signers")
	}
}
