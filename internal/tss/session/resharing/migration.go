package resharing

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	utxoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/utxo"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"golang.org/x/sync/errgroup"
)

type MigrationSession struct {
	params Params
	newKey *ecdsa.PublicKey

	self           tss.LocalSignParty
	sessionManager *p2p.SessionManager
	chains         chain.Repository

	logger *logan.Entry
}

func NewMigrationSession(
	params Params,
	newKey *ecdsa.PublicKey,
	self tss.LocalSignParty,
	sessionManager *p2p.SessionManager,
	chains chain.Repository,
	logger *logan.Entry,
) *MigrationSession {
	return &MigrationSession{
		params: params,
		newKey: newKey,

		self:           self,
		sessionManager: sessionManager,
		chains:         chains,

		logger: logger,
	}
}

func (s *MigrationSession) Run(ctx context.Context) error {
	state := resharingTypes.InitializeState(s.params.Epoch, s.params.Threshold, s.params.StartTime, 0, s.self.Account)
	state.SessionStartTime = s.params.StartTime
	state.NewPubKey = s.newKey

	var managers []*resharingTypes.HandlerManager
	for _, ch := range s.chains.Clients() {
		if ch.Type() != chain.TypeBitcoin {
			continue
		}

		manager := resharingTypes.NewHandlerManager(
			utxo.NewHandler(s.self, s.params.Parties, ch.(utxoclient.Client), s.sessionManager, s.logger),
			state,
			s.logger.WithField("component", fmt.Sprintf("utxo_migration_manager_%s", ch.ChainId())),
		)

		managers = append(managers, manager)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, manager := range managers {
		eg.Go(func() error { return manager.Manage(egCtx, state.SessionStartTime) })
	}

	return errors.Wrap(eg.Wait(), "failed to successfully run resharing session managers")
}
