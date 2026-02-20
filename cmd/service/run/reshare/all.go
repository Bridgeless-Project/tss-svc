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

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		if err := session.Run(ctx); err != nil {
			return errors.Wrap(err, "failed to run resharing session")
		}

		return nil
	},
}
