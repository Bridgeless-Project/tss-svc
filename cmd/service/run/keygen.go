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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
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

		// TODO use the protocol id
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

		connectionManager := p2p.NewConnectionManager(
			parties,
			p2p.PartyStatus_PS_KEYGEN,
			cfg.Log().WithField("component", "connection_manager"),
		)

		frostSeession := keygenSession.NewSession(
			tss.LocalKeygenParty{
				PreParams: preParams,
				Address:   account.CosmosAddress(),
				Threshold: cfg.TssSessionParams().Threshold,
			},
			parties,
			cfg.TssSessionParams(),
			connectionManager.GetReadyCount,
			cfg.Log().WithField("component", "keygen_session"),
			tss.ProtocolID_FROST,
			curve.Secp256k1{}, // TODO implement custom curve for ZCash
		)

		//ecdsaSeession := keygenSession.NewSession(
		//	tss.LocalKeygenParty{
		//		PreParams: *preParams,
		//		Address:   account.CosmosAddress(),
		//		Threshold: cfg.TssSessionParams().Threshold,
		//	},
		//	parties,
		//	cfg.TssSessionParams(),
		//	connectionManager.GetReadyCount,
		//	cfg.Log().WithField("component", "keygen_session"),
		//	tss.ProtocolID_ECDSA,
		//	curve.Secp256k1{},
		//)
		sessionManager := p2p.NewSessionManager(frostSeession)

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

			if err := frostSeession.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run keygen session")
			}
			result, err := frostSeession.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain frost keygen session result")
			}

			cfg.Log().Info("keygen session successfully completed")

			return storeKeygenResult(result, storage)
		})

		//	errGroup.Go(func() error {
		//		defer cancel()
		//
		//		if err := ecdsaSeession.Run(ctx); err != nil {
		//			return errors.Wrap(err, "failed to run keygen session")
		//		}
		//		result, err := ecdsaSeession.WaitFor()
		//		if err != nil {
		//			return errors.Wrap(err, "failed to obtain ecdsa keygen session result")
		//		}
		//
		//		cfg.Log().Info("keygen session successfully completed")
		//
		//		return storeKeygenResult(result, storage)
		//	})

		return errGroup.Wait()
	},
}

func storeKeygenResult(result interface{}, storage secrets.Storage) error {
	if localData, ok := result.(*tss.LocalPartyData); ok {
		result = localData.GetData()
	}

	utils.OutputType = "vault"
	switch utils.OutputType {
	case "console":
		raw, err := json.Marshal(result)
		if err != nil {
			return errors.Wrap(err, "failed to marshal keygen result")
		}
		fmt.Println("raw: ", string(raw))
	case "file":
		fmt.Println("file")
		raw, err := json.Marshal(result)
		if err != nil {
			return errors.Wrap(err, "failed to marshal keygen result")
		}
		if err = os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write keygen result to file")
		}
	case "vault":
		fmt.Println("saving keygen result to vault...")
		if err := storage.SaveTssShare(result); err != nil {
			return errors.Wrap(err, "failed to save keygen result to vault")
		}
	default:
		fmt.Println("unknown output type:", utils.OutputType)
	}

	fmt.Println("done")

	return nil
}
