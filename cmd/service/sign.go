package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	tss_lib "github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	"github.com/hyle-team/tss-svc/internal/config"
	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/hyle-team/tss-svc/internal/tss"
	"github.com/hyle-team/tss-svc/internal/tss/consensus"
	"github.com/hyle-team/tss-svc/internal/tss/session"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	utils.RegisterOutputFlags(signCmd)
}

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
		//if len(dataToSign) == 0 {
		//	return errors.Wrap(errors.New("empty data to-sign"), "invalid data")
		//}

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

		consensus := session.NewConsensusSession(consensus.LocalParams{
			PartyStatus: consensus.Signer,
			Address:     account.CosmosAddress(),
		}, cfg.TSSParams().ConsensusParams(), cfg.Log().WithField("component", "consensus_session"), []byte(dataToSign), connectionManager.GetReadyCount, cfg.Parties(), form, validate, account.CosmosAddress(), chain.Chain{}, "Bitcoin", selector)

		sessionManager := p2p.NewSessionManager(consensus)
		errGroup.Go(func() error {
			server := p2p.NewServer(cfg.GRPCListener(), sessionManager)
			server.SetStatus(p2p.PartyStatus_SIGNING)
			return server.Run(ctx)
		})

		errGroup.Go(func() error {
			defer cancel()

			if err := consensus.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run consensus session")
			}
			data, parties, err := consensus.WaitFor()
			cfg.Log().Info("consensus session finished ", "parties ", parties, "err ", err, "data ", data)
			if data == nil || parties == nil {
				if err != nil {
					return errors.Wrap(err, "consensus error")
				}
				cfg.Log().Info("not a signer")
				return nil
			}
			cfg.Log().Info("Try to start signing session")
			connectionManager := p2p.NewConnectionManager(parties, p2p.PartyStatus_SIGNING, cfg.Log().WithField("component", "connection_manager"))
			signSession := session.NewDefaultSigningSession(
				tss.LocalSignParty{
					Address:   account.CosmosAddress(),
					Data:      localSaveData,
					Threshold: cfg.TSSParams().SigningSessionParams().Threshold,
				},
				cfg.TSSParams().SigningSessionParams(),
				cfg.Log().WithField("component", "signing_session"),
				parties,
				data,
				connectionManager.GetReadyCount,
			)
			sessionManager.Add(signSession)
			signSession.AddStartTime(10 * time.Second)
			if err := signSession.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run signing session")
			}
			result, err := signSession.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain signing session result")
			}

			cfg.Log().Info("signing session successfully completed")
			err = saveSigningResult(result)
			if err != nil {
				return errors.Wrap(err, "failed to save signing result")
			}
			verifySignature(localSaveData, data, result, cfg)
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

func verifySignature(localData *keygen.LocalPartySaveData, inputData []byte, signature *common.SignatureData, cfg config.Config) {
	if utils.IsVerifyNeeded {
		pk := ecdsa.PublicKey{
			Curve: tss_lib.EC(),
			X:     localData.ECDSAPub.X(),
			Y:     localData.ECDSAPub.Y(),
		}
		ok := ecdsa.Verify(&pk, big.NewInt(0).SetBytes(inputData).Bytes(), big.NewInt(0).SetBytes(signature.R), big.NewInt(0).SetBytes(signature.S))

		if ok {
			cfg.Log().Info("signature is valid")
		}
		if !ok {
			cfg.Log().Warn("signature is invalid")
		}
	}
}

func form(data []byte) ([]byte, error) {
	return data, nil
}

func validate(data []byte) (bool, error) {
	return true, nil
}

func selector(chainId string, data []byte) ([]byte, error) {
	if data == nil {
		return nil, errors.New("no data")
	}
	return data, nil
}
