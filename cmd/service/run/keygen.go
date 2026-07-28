package run

import (
	"context"
	"fmt"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets/sharefile"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsaTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	frostTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/frost"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	keygenSession "github.com/Bridgeless-Project/tss-svc/internal/tss/session/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"golang.org/x/sync/errgroup"
)

func init() {
	keygenCmd.Flags().StringVarP(&keygenOutputType, "output", "o", "vault", "Output type: vault, file, or console")
	keygenCmd.Flags().StringVar(&keygenFilePath, "path", "tss-share.json", "Base path for protocol-specific share files")
	keygenCmd.Flags().BoolVar(&unsafeKeygenConsole, "unsafe-console", false, "Allow private TSS shares to be printed to the terminal")
	utils.RegisterConfigFlag(keygenCmd)
}

const KeygensCount = 2

var (
	keygenOutputType    string
	keygenFilePath      string
	unsafeKeygenConsole bool
)

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generates a new keypair using TSS",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if keygenOutputType != "console" && keygenOutputType != "file" && keygenOutputType != "vault" {
			return errors.New("invalid output type")
		}
		if keygenOutputType == "console" && !unsafeKeygenConsole {
			return errors.New("printing private TSS shares requires --unsafe-console")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := cfg.SecretsStorage()

		preParams := ecdsaTss.NewEcdsaPreParams()
		if err := storage.GetKeygenPreParams(preParams); err != nil {
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

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		errGroup, ctx := errgroup.WithContext(ctx)

		frostSeession := keygenSession.NewSession(
			tss.LocalKeygenParty{
				PreParams: frostTss.NewFrostPreParams(),
				Account:   *account,
				Address:   account.CosmosAddress(),
				Threshold: cfg.TssSessionParams().Threshold,
			},
			parties,
			cfg.TssSessionParams(),
			cfg.Log().WithField("component", "keygen_session"),
			curve.Secp256k1{}, // TODO implement custom curve for ZCash
		)

		ecdsaSeession := keygenSession.NewSession(
			tss.LocalKeygenParty{
				PreParams: preParams,
				Account:   *account,
				Address:   account.CosmosAddress(),
				Threshold: cfg.TssSessionParams().Threshold,
			},
			parties,
			cfg.TssSessionParams(),
			cfg.Log().WithField("component", "keygen_session"),
			curve.Secp256k1{},
		)
		sessionManager := p2p.NewSessionManager(frostSeession, ecdsaSeession)

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

		resultChan := make(chan tss.Share, KeygensCount)
		startChan := make(chan struct{})
		errGroup.Go(func() error {
			if err := waitForKeygenStart(ctx, cfg.TssSessionParams().StartTime); err != nil {
				return err
			}
			close(startChan)
			return nil
		})
		errGroup.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-startChan:
			}

			if err := frostSeession.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run keygen session")
			}
			result, err := frostSeession.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain frost keygen session result")
			}

			cfg.Log().Info("frost keygen session successfully completed")
			resultChan <- result
			return nil
		})

		errGroup.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-startChan:
			}

			if err := ecdsaSeession.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run keygen session")
			}
			result, err := ecdsaSeession.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain ecdsa keygen session result")
			}

			cfg.Log().Info("ecdsa keygen session successfully completed")
			resultChan <- result
			return nil
		})

		errGroup.Go(func() error {
			results := make([]tss.Share, 0, KeygensCount)
			for {
				select {
				case <-ctx.Done():
					return nil
				case result := <-resultChan:
					results = append(results, result)
					if len(results) == KeygensCount {
						if err := storeKeygenResults(results, storage); err != nil {
							return errors.Wrap(err, "failed to store keygen shares")
						}
						cancel()
						return nil
					}
				}
			}
		})

		return errGroup.Wait()
	},
}

func waitForKeygenStart(ctx context.Context, start time.Time) error {
	return errors.Wrap(session.WaitUntil(ctx, start), "keygen interrupted before its configured start time")
}

func storeKeygenResults(results []tss.Share, storage secrets.Storage) error {
	if len(results) != KeygensCount {
		return errors.Errorf("expected %d keygen results, got %d", KeygensCount, len(results))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Protocol() < results[j].Protocol() })
	rawResults := make(map[tss.ProtocolType][]byte, len(results))
	for _, result := range results {
		if result == nil {
			return errors.New("nil keygen result")
		}
		raw, err := result.Marshal()
		if err != nil {
			return errors.Wrapf(err, "failed to marshal %s keygen result", result.Protocol())
		}
		rawResults[result.Protocol()] = raw
	}
	if len(rawResults) != len(results) {
		return errors.New("duplicate keygen protocol result")
	}

	switch keygenOutputType {
	case "console":
		if !unsafeKeygenConsole {
			return errors.New("printing private TSS shares requires --unsafe-console")
		}
		for _, result := range results {
			fmt.Printf("%s: %s\n", result.Protocol(), rawResults[result.Protocol()])
		}
	case "file":
		_, err := sharefile.WriteSet(keygenFilePath, rawResults)
		return err
	case "vault":
		// Do not publish either protocol until both ceremonies have completed and
		// both results have serialized successfully.
		return secrets.SaveTssShareSet(storage, results)
	default:
		return errors.Errorf("unknown output type: %s", keygenOutputType)
	}

	return nil
}
