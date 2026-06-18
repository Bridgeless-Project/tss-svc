package reshare

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/repository"
	coreConnector "github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var reshareAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Command to run all necessary operations for key resharing",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		params := cfg.ResharingParams()
		oldParams := cfg.TssSessionParams()
		oldParties := cfg.Parties()
		secrets := cfg.SecretsStorage()
		clients := cfg.Clients()
		clientsRepo := repository.NewClientsRepository(clients)
		sessionManager := p2p.NewSessionManager()
		logger := cfg.Log()
		account, err := secrets.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
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

		serverCert, err := secrets.GetLocalPartyTlsCertificate()
		if err != nil {
			return errors.Wrap(err, "failed to get local party TLS certificate")
		}

		session := resharing.NewSession(
			params,
			oldParties,
			oldParams,
			secrets,
			sessionManager,
			core,
			clientsRepo,
			logger.WithField("component", "resharing_session"),
		)

		var (
			termCtx, cancel = signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
			eg, ctx         = errgroup.WithContext(termCtx)
		)

		eg.Go(func() error {
			defer cancel()

			logger.Info("resharing started")
			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "resharing failed")
			}
			logger.Info("resharing finished")

			return nil
		})
		eg.Go(func() error {
			server := p2p.NewServer(
				cfg.P2pGrpcListener(),
				sessionManager,
				p2p.MergeParties(append(oldParties, params.Parties...)...),
				*serverCert,
				cfg.Log().WithField("component", "p2p_server"),
			)
			server.SetStatus(p2p.PartyStatus_PS_RESHARE)

			return server.Run(ctx)
		})

		return eg.Wait()
	},
}
