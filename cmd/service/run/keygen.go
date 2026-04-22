package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	keygenSession "github.com/Bridgeless-Project/tss-svc/internal/tss/session/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	utils.RegisterOutputFlags(keygenCmd)
}

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generates a new keypair using TSS",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !utils.OutputValid() {
			return errors.New("invalid output type")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := cfg.SecretsStorage()
		preParams, err := storage.GetKeygenPreParams()
		if err != nil {
			return errors.Wrap(err, "failed to get keygen pre-parameters")
		}
		account, err := storage.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}
		cert, err := storage.GetLocalPartyTlsCertificate()
		if err != nil {
			return errors.Wrap(err, "failed to get local party TLS certificate")
		}
		parties := cfg.Parties()

		errGroup := new(errgroup.Group)
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		session := keygenSession.NewSession(
			tss.LocalKeygenParty{
				PreParams: *preParams,
				Address:   account.CosmosAddress(),
				Threshold: cfg.TssSessionParams().Threshold,
			},
			parties,
			cfg.TssSessionParams(),
			cfg.Log().WithField("component", "keygen_session"),
		)

		sessionManager := p2p.NewSessionManager(session)

		errGroup.Go(func() error {
			server := p2p.NewServer(
				cfg.P2pGrpcListener(),
				sessionManager,
				parties,
				*cert,
				cfg.Log().WithField("component", "p2p_server"),
			)
			server.SetStatus(p2p.PartyStatus_PS_KEYGEN)
			return server.Run(ctx)
		})

		errGroup.Go(func() error {
			defer cancel()

			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run keygen session")
			}
			result, err := session.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain keygen session result")
			}

			cfg.Log().Info("keygen session successfully completed")

			return storeKeygenResult(result, storage)
		})

		return errGroup.Wait()
	},
}

func storeKeygenResult(result *keygen.LocalPartySaveData, storage secrets.Storage) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal keygen result")
	}

	switch utils.OutputType {
	case "console":
		fmt.Println(string(raw))
	case "file":
		if err = os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write keygen result to file")
		}
	case "vault":
		if err = storage.SaveTssShare(result); err != nil {
			return errors.Wrap(err, "failed to save keygen result to vault")
		}
	}

	return nil
}
