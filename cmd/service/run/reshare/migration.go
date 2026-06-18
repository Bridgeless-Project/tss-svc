package reshare

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/repository"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var reshareMigrationCmd = &cobra.Command{
	Use:   "migration",
	Short: "Command to run final funds migration from the old to the new key after resharing",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		params := cfg.ResharingParams()
		secrets := cfg.SecretsStorage()
		clients := cfg.Clients()
		clientsRepo := repository.NewClientsRepository(clients)
		sessionManager := p2p.NewSessionManager()
		logger := cfg.Log()

		account, err := secrets.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}
		oldKeyShare, err := secrets.GetTemporaryTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get old key share")
		}

		self := tss.LocalSignParty{
			Account:   *account,
			Share:     oldKeyShare,
			Threshold: int(params.Threshold),
		}

		core, err := coreConnector.NewConnector(
			*account,
			cfg.CoreConnectorConfig().Connection,
			cfg.CoreConnectorConfig().Settings,
			logger.WithField("component", "core_connector"),
		)
		if err != nil {
			return errors.Wrap(err, "failed to create core connector")
		}
		rawKey, err := core.GetEpochPubKey(params.Epoch)
		if err != nil {
			return errors.Wrap(err, "failed to get new public key from core")
		}
		newKey := bridge.MustDecodePubkey(rawKey)

		serverCert, err := secrets.GetLocalPartyTlsCertificate()
		if err != nil {
			return errors.Wrap(err, "failed to get local party TLS certificate")
		}

		session := resharing.NewMigrationSession(
			params,
			newKey,
			self,
			sessionManager,
			clientsRepo,
			logger.WithField("component", "migration_session"),
		)

		var (
			termCtx, cancel = signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
			eg, ctx         = errgroup.WithContext(termCtx)
		)

		eg.Go(func() error {
			defer cancel()

			logger.Info("migration started")
			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "resharing failed")
			}
			logger.Info("migration finished")

			return nil
		})
		eg.Go(func() error {
			server := p2p.NewServer(
				cfg.P2pGrpcListener(),
				sessionManager,
				params.Parties,
				*serverCert,
				cfg.Log().WithField("component", "p2p_server"),
			)
			server.SetStatus(p2p.PartyStatus_PS_RESHARE)

			return server.Run(ctx)
		})

		return eg.Wait()
	},
}
