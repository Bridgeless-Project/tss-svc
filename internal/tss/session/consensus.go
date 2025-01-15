package session

import (
	"context"
	"fmt"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss/consensus"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
	"sync"
	"time"
)

type ConsensusParams struct {
	Id        string
	StartTime time.Time
	Threshold int
}

type ConsensusSession struct {
	wg     *sync.WaitGroup
	logger *logan.Entry

	params                ConsensusParams
	connectedPartiesCount func() int
	partiesCount          int
	localPartyAddr        core.Address

	data          []byte
	resultSigners []p2p.Party
	err           error

	consensus interface {
		Run(ctx context.Context)
		WaitFor() ([]byte, []p2p.Party, error)
		Receive(sender core.Address, data *p2p.TssData, reqType p2p.RequestType)
	}
}

func NewConsensusSession(self consensus.LocalParams, params ConsensusParams, logger *logan.Entry, data []byte, counterFunc func() int, parties []p2p.Party, formData func([]byte) ([]byte, error), validateData func([]byte) (bool, error), local core.Address, metadata chain.ChainMetadata, chainId string, dataSelector func(string, []byte) ([]byte, error)) *ConsensusSession {
	return &ConsensusSession{
		wg:                    &sync.WaitGroup{},
		params:                params,
		logger:                logger,
		data:                  data,
		localPartyAddr:        local,
		connectedPartiesCount: counterFunc,
		partiesCount:          len(parties),
		consensus:             consensus.NewConsensus(self, parties, logger, params.Id, data, formData, validateData, params.Threshold, metadata, chainId, dataSelector),
	}
}

func (s *ConsensusSession) Run(ctx context.Context) error {
	runDelay := time.Until(s.params.StartTime)
	if runDelay <= 0 {
		return errors.New("target time is in the past")
	}

	s.logger.Info(fmt.Sprintf("consensus session will start in %s", runDelay))

	select {
	case <-ctx.Done():
		s.logger.Info("consensus session cancelled")
		return nil
	case <-time.After(runDelay):
	}

	if s.connectedPartiesCount() != s.partiesCount {
		return errors.New("cannot start consensus session: not all parties connected")
	}

	s.wg.Add(1)
	go s.run(ctx)
	return nil
}

func (c *ConsensusSession) run(ctx context.Context) {
	defer c.wg.Done()

	boundedCtx, cancel := context.WithTimeout(ctx, BoundaryConsensusSession)
	defer cancel()

	c.consensus.Run(boundedCtx)
	c.data, c.resultSigners, c.err = c.consensus.WaitFor()

	if err := boundedCtx.Err(); err != nil {
		c.err = err
	} else {
		c.err = errors.Wrap(c.err, "consensus session timed out")
	}
	c.logger.Info("consensus session finished")

}

func (c *ConsensusSession) WaitFor() ([]byte, []p2p.Party, error) {
	c.wg.Wait()
	return c.data, c.resultSigners, c.err
}

func (c *ConsensusSession) Id() string {
	return c.params.Id
}
func (c *ConsensusSession) Receive(request *p2p.SubmitRequest) error {
	if request == nil || request.Data == nil {
		return errors.New("nil request")
	}
	if request.Type != p2p.RequestType_ACK && request.Type != p2p.RequestType_NACK && request.Type != p2p.RequestType_DATA_TO_SIGN && request.Type != p2p.RequestType_NO_DATA_TO_SIGN && request.Type != p2p.RequestType_SIGNER_NOTIFY && request.Type != p2p.RequestType_RAW_DATA {
		return errors.New("invalid request type: " + request.Type.String())
	}

	data := &p2p.TssData{}
	if err := request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal TSS request data")
	}
	sender, err := core.AddressFromString(request.Sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}
	c.consensus.Receive(sender, data, request.Type)

	return nil
}

// RegisterIdChangeListener is a no-op for Consensus
func (c *ConsensusSession) RegisterIdChangeListener(func(oldId string, newId string)) {
}
