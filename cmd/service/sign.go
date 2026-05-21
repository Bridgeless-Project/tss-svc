package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/secrets"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/bnb-chain/tss-lib/v3/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	utils.RegisterOutputFlags(signCmd)
	registerSignCmdFlags(signCmd)
}

const (
	signProtocolECDSA = "ecdsa"
	signProtocolFROST = "frost"
)

var (
	verify       bool
	signProtocol string
)

func registerSignCmdFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&verify, "verify", true, "Whether to additionally verify the signature")
	cmd.Flags().StringVar(&signProtocol, "protocol", signProtocolECDSA, "TSS protocol to use for signing: ecdsa or frost")
}

var signCmd = &cobra.Command{
	Use:   "sign [data-hex]",
	Short: "Signs the given hex-decoded data using TSS",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !utils.OutputValid() {
			return errors.New("invalid output type")
		}
		return validateSignProtocol(signProtocol)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to read config from flags")
		}

		rawData := args[0]
		if !strings.HasPrefix(rawData, bridge.HexPrefix) {
			rawData = bridge.HexPrefix + rawData
		}

		dataToSign := hexutil.MustDecode(rawData)
		if len(dataToSign) == 0 {
			return errors.Wrap(errors.New("empty data to-sign"), "invalid data")
		}

		storage := cfg.SecretsStorage()
		account, err := storage.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}
		shares, err := storage.GetTssShares()
		if err != nil {
			return errors.Wrap(err, "failed to get local shares")
		}
		cert, err := storage.GetLocalPartyTlsCertificate()
		if err != nil {
			return errors.Wrap(err, "failed to get local party TLS certificate")
		}
		parties := cfg.Parties()

		errGroup := new(errgroup.Group)
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		localParty, err := localSignPartyForProtocol(*account, shares, cfg.TssSessionParams().Threshold, signProtocol)
		if err != nil {
			return err
		}

		session := signing.NewSession(
			localParty,
			signing.SessionParams{
				Params:      cfg.TssSessionParams(),
				SigningData: dataToSign,
			},
			parties,
			cfg.Log().WithField("component", "signing_session"),
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
			server.SetStatus(p2p.PartyStatus_PS_SIGN)
			return server.Run(ctx)
		})

		errGroup.Go(func() error {
			defer cancel()

			select {
			case <-ctx.Done():
				return errors.New("resharing session was interrupted before it started")
			case <-time.After(time.Until(cfg.TssSessionParams().StartTime)):
				break
			}

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
				if valid := verifySigningResult(localParty, signProtocol, dataToSign, result); !valid {
					return errors.New("signature verification failed")
				}
				cfg.Log().Info("Signature verification passed")

			}

			return nil
		})
		return errGroup.Wait()
	},
}

func validateSignProtocol(protocol string) error {
	switch protocol {
	case signProtocolECDSA, signProtocolFROST:
		return nil
	default:
		return errors.Errorf("unsupported signing protocol: %s", protocol)
	}
}

func localSignPartyForProtocol(account core.Account, shares *secrets.TssShares, threshold int, protocol string) (tss.LocalSignParty, error) {
	if shares == nil {
		return tss.LocalSignParty{}, errors.New("missing TSS shares")
	}

	localParty := tss.LocalSignParty{
		Account:   account,
		Threshold: threshold,
	}

	switch protocol {
	case signProtocolECDSA:
		share, err := tss.ECDSAShare(shares.Share)
		if err != nil {
			return tss.LocalSignParty{}, errors.Wrap(err, "ECDSA share is required")
		}
		localParty.Share = share
	case signProtocolFROST:
		share, err := tss.FrostShare(shares.FrostShare)
		if err != nil {
			return tss.LocalSignParty{}, errors.Wrap(err, "FROST share is required")
		}
		localParty.FrostShare = share
	default:
		return tss.LocalSignParty{}, errors.Errorf("unsupported signing protocol: %s", protocol)
	}

	return localParty, nil
}

func verifySigningResult(localParty tss.LocalSignParty, protocol string, data []byte, result *common.SignatureData) bool {
	switch protocol {
	case signProtocolECDSA:
		if localParty.Share == nil {
			return false
		}
		return tss.Verify(localParty.Share.ECDSAPub.ToECDSAPubKey(), data, result)
	case signProtocolFROST:
		pubKey, err := tss.FrostPubKey(localParty.FrostShare)
		if err != nil {
			return false
		}
		return tss.VerifyFrost(pubKey, data, result)
	default:
		return false
	}
}

func saveSigningResult(result *common.SignatureData) error {
	signature := hexutil.Encode(append(result.Signature, result.SignatureRecovery...))

	switch utils.OutputType {
	case "console":
		fmt.Println(signature)
	case "file":
		raw, err := json.Marshal(signature)
		if err != nil {
			return errors.Wrap(err, "failed to marshal signing result")
		}
		if err = os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write signing result to file")
		}
	}
	return nil
}
