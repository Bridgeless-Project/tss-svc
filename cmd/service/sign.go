package service

import (
	"context"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os/signal"
	"syscall"
)

var signCmd = &cobra.Command{
	Use:  "sign [data-string]",
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !utils.OutputValid() {
			return errors.New("invalid output type")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to read config from flags")
		}

		dataToSign := args[0]
		if len(dataToSign) == 0 {
			return errors.Wrap(errors.New("empty data to-sign"), "invalid data")
		}

		storage := vault.NewStorage(cfg.VaultClient())

		account, err := storage.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}
		localSaveData, err := storage.GetTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get local share")
		}

		errGroup := new(errgroup.Group)
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		connectionManager := p2p.NewConnectionManager(cfg.Parties(), p2p.PartyStatus_SIGNING, cfg.Log().WithField("component", "connection_manager"))

		session := session.NewDefaultSigningSession(
			tss.LocalSignParty{
				Address:   account.CosmosAddress(),
				Data:      localSaveData,
				Threshold: cfg.TSSParams().SigningSessionParams().Threshold,
			},
			cfg.TSSParams().SigningSessionParams(),
			cfg.Log().WithField("component", "signing_session"),
			cfg.Parties(),
			dataToSign,
			connectionManager.GetReadyCount,
		)

		sessionManager := p2p.NewSessionManager(session)
		errGroup.Go(func() error {
			server := p2p.NewServer(cfg.GRPCListener(), sessionManager)
			server.SetStatus(p2p.PartyStatus_SIGNING)
			return server.Run(ctx)
		})

		errGroup.Go(func() error {
			defer cancel()

			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run signing session")
			}
			result, err := session.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain signing session result")
			}

			cfg.Log().Info("signing session successfully completed. Signature: ", result.String())

			return nil
		})

		return errGroup.Wait()
	},
}
