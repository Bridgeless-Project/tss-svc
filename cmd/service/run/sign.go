package run

import (
	"context"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	tsslib "github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os/signal"
	"strconv"
	"syscall"
)

var signCmd = &cobra.Command{
	Use:  "sign [data-string] [threshold]",
	Args: cobra.ExactArgs(2),
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
		arg2 := args[1]
		threshoold, err := strconv.Atoi(arg2)
		if err != nil {
			return errors.Wrap(err, "invalid threshold")
		}

		storage := vault.NewStorage(cfg.VaultClient())

		// Configuring local data for LocalSignParty
		localData := keygen.NewLocalPartySaveData(len(cfg.Parties()))
		var partyIds []*tsslib.PartyID
		for _, party := range cfg.Parties() {
			partyIds = append(partyIds, party.Identifier())
		}
		localSaveData := keygen.BuildLocalSaveDataSubset(localData, tsslib.SortPartyIDs(partyIds))
		account, err := storage.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}

		errGroup := new(errgroup.Group)
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		connectionManager := p2p.NewConnectionManager(cfg.Parties(), p2p.PartyStatus_SIGNING, cfg.Log().WithField("component", "connection_manager"))

		session := session.NewSigningSession(
			tss.LocalSignParty{
				Address:   account.CosmosAddress(),
				Data:      &localSaveData,
				Threshold: threshoold,
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
