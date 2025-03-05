package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	utils.RegisterOutputFlags(signCmd)
	registerSignCmdFlags(signCmd)
}

var verify bool

func registerSignCmdFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&verify, "verify", true, "Whether to additionally verify the signature")
}

var signCmd = &cobra.Command{
	Use:   "sign [data-string]",
	Short: "Signs the given data using TSS",
	Args:  cobra.ExactArgs(1),
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

		connectionManager := p2p.NewConnectionManager(cfg.Parties(), p2p.PartyStatus_PS_SIGN, cfg.Log().WithField("component", "connection_manager"))

		session := session.NewDefaultSigningSession(
			tss.LocalSignParty{
				Address:   account.CosmosAddress(),
				Share:     localSaveData,
				Threshold: cfg.TssSessionParams().Threshold,
			},
			session.DefaultSigningSessionParams{
				SessionParams: cfg.TssSessionParams(),
				SigningData:   []byte(dataToSign),
			},
			cfg.Parties(),
			connectionManager.GetReadyCount,
			cfg.Log().WithField("component", "signing_session"),
		)

		sessionManager := p2p.NewSessionManager(session)
		errGroup.Go(func() error {
			server := p2p.NewServer(cfg.P2pGrpcListener(), sessionManager, cfg.Log().WithField("component", "p2p_server"))
			server.SetStatus(p2p.PartyStatus_PS_SIGN)
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

			cfg.Log().Info("Signing session successfully completed")
			if err = saveSigningResult(result); err != nil {
				return errors.Wrap(err, "failed to save signing result")
			}

			if verify {
				valid := tss.Verify(localSaveData, []byte(dataToSign), result)
				cfg.Log().Infof("Verified signature valid: %t", valid)
			}

			return nil
		})
		return errGroup.Wait()
	},
}

func saveSigningResult(result *common.SignatureData) error {
	signature := hexutil.Encode(append(result.Signature, result.SignatureRecovery...))
	raw, err := json.Marshal(signature)
	if err != nil {
		return errors.Wrap(err, "failed to marshal signing result")
	}

	switch utils.OutputType {
	case "console":
		fmt.Println(string(raw))
	case "file":
		if err = os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write signing result to file")
		}
	}
	return nil
}
