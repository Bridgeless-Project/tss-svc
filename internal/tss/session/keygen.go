package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ p2p.TssSession = &KeygenSession{}

type KeygenSessionParams struct {
	Id        string
	StartTime time.Time
}

type KeygenSession struct {
	params KeygenSessionParams
	wg     *sync.WaitGroup

	connectedPartiesCount func() int
	partiesCount          int

	keygenParty interface {
		Run(ctx context.Context)
		WaitFor() *keygen.LocalPartySaveData
		Receive(sender core.Address, data *p2p.TssData)
	}

	result *keygen.LocalPartySaveData
	err    error

	logger *logan.Entry
}

func NewKeygenSession(
	self tss.LocalKeygenParty,
	parties []p2p.Party,
	params KeygenSessionParams,
	connectedPartiesCountFunc func() int,
	logger *logan.Entry,
) *KeygenSession {
	return &KeygenSession{
		params:                params,
		wg:                    &sync.WaitGroup{},
		connectedPartiesCount: connectedPartiesCountFunc,
		partiesCount:          len(parties),
		keygenParty:           tss.NewKeygenParty(self, parties, params.Id, logger.WithField("component", "keygen_party")),
		logger:                logger,
	}
}

func (s *KeygenSession) Run(ctx context.Context) error {
	runDelay := time.Until(s.params.StartTime)
	if runDelay <= 0 {
		return errors.New("target time is in the past")
	}

	s.logger.Info(fmt.Sprintf("keygen session will start in %s", runDelay))

	select {
	case <-ctx.Done():
		s.logger.Info("keygen session cancelled")
		return nil
	case <-time.After(runDelay):
	}

	if s.connectedPartiesCount() != s.partiesCount {
		return errors.New("cannot start keygen session: not all parties connected")
	}

	s.wg.Add(1)
	go s.run(ctx)
	return nil
}

func (s *KeygenSession) run(ctx context.Context) {
	defer s.wg.Done()

	boundedCtx, cancel := context.WithTimeout(ctx, BoundaryKeygenSession)
	defer cancel()

	s.keygenParty.Run(boundedCtx)
	s.result = s.keygenParty.WaitFor()
	s.logger.Info("keygen session finished")
	if s.result != nil {
		return
	}

	if err := boundedCtx.Err(); err != nil {
		s.err = err
	} else {
		s.err = errors.New("keygen session error occurred")
	}
}

func (s *KeygenSession) WaitFor() (*keygen.LocalPartySaveData, error) {
	s.wg.Wait()
	return s.result, s.err
}

func (s *KeygenSession) Id() string {
	return s.params.Id
}

func (s *KeygenSession) Receive(request *p2p.SubmitRequest) error {
	if request.Type != p2p.RequestType_KEYGEN {
		return errors.New("invalid request type")
	}

	var data *p2p.TssData

	if err := request.Data.UnmarshalTo(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal TSS request data")
	}

	sender, _ := core.AddressFromString(request.Sender)

	// TODO: add better error handling?
	s.keygenParty.Receive(sender, data)
	return nil
}

// RegisterIdChangeListener is a no-op for KeygenSession
func (s *KeygenSession) RegisterIdChangeListener(func(oldId, newId string)) {}
