package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/taurusgroup/multi-party-sig/pkg/math/curve"
	"github.com/taurusgroup/multi-party-sig/pkg/taproot"
	frostkeygen "github.com/taurusgroup/multi-party-sig/protocols/frost/keygen"
	"golang.org/x/sync/errgroup"
)

func init() {
	utils.RegisterOutputFlags(signCmd)
	registerSignCmdFlags(signCmd)
}

var verify bool

const defaultFrostMockMessage = "frost mock string"

func registerSignCmdFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&verify, "verify", true, "Whether to additionally verify the signature")
}

var signCmd = &cobra.Command{
	Use:   "sign [data-hex]",
	Short: "Signs the given hex-decoded data using FROST TSS",
	Args:  cobra.MaximumNArgs(1),
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

		message, messageToHash, err := frostSigningInput(args)
		if err != nil {
			return err
		}
		messageHash := sha256.Sum256(messageToHash)
		dataToSign := messageHash[:]
		if len(dataToSign) == 0 {
			return errors.Wrap(errors.New("empty data to-sign"), "invalid data")
		}

		storage := cfg.SecretsStorage()
		account, err := storage.GetCoreAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get core account")
		}
		share, err := storage.GetTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get local share")
		}
		frostShare, ok := share.(*frostkeygen.Config)
		if !ok {
			return errors.Errorf("expected FROST share from vault, got %T", share)
		}
		pubKey, err := frostPubKey(frostShare)
		if err != nil {
			return errors.Wrap(err, "failed to get FROST public key")
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
			parties, p2p.PartyStatus_PS_SIGN, cfg.Log().WithField("component", "connection_manager"),
		)

		session := signing.NewDefaultSession(
			tss.LocalSignParty{
				Account:    *account,
				FrostShare: frostShare,
				Threshold:  cfg.TssSessionParams().Threshold,
			},
			signing.DefaultSessionParams{
				Params:      cfg.TssSessionParams(),
				SigningData: dataToSign,
			},
			parties,
			connectionManager.GetReadyCount,
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

			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run signing session")
			}
			result, err := session.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain signing session result")
			}

			cfg.Log().Info("Signing session successfully completed")
			output := newFrostSigningOutput(message, dataToSign, pubKey, result)
			if verify {
				output.Test = taproot.PublicKey(pubKey).Verify(taproot.Signature(result.Signature), dataToSign)
				if !output.Test {
					return errors.New("signature verification failed")
				}
				cfg.Log().Info("Signature verification passed")
			}

			if err = saveSigningResult(output); err != nil {
				return errors.Wrap(err, "failed to save signing result")
			}

			return nil
		})
		return errGroup.Wait()
	},
}

type frostSigningOutput struct {
	Protocol    string `json:"protocol"`
	Curve       string `json:"curve"`
	Message     string `json:"message"`
	MessageHash string `json:"message_hash"`
	PubKey      string `json:"pubkey"`
	Signature   string `json:"signature"`
	Test        bool   `json:"test"`
}

func frostSigningInput(args []string) (string, []byte, error) {
	if len(args) == 0 {
		return defaultFrostMockMessage, []byte(defaultFrostMockMessage), nil
	}

	rawData := args[0]
	if !strings.HasPrefix(rawData, bridge.HexPrefix) {
		rawData = bridge.HexPrefix + rawData
	}

	data, err := hexutil.Decode(rawData)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to decode hex signing data")
	}
	if len(data) == 0 {
		return "", nil, errors.Wrap(errors.New("empty data to-sign"), "invalid data")
	}

	return string(data), data, nil
}

func frostPubKey(share *frostkeygen.Config) ([]byte, error) {
	if share == nil {
		return nil, errors.New("nil FROST share")
	}

	publicKey, ok := share.PublicKey.(*curve.Secp256k1Point)
	if !ok {
		return nil, errors.New("FROST public key is not secp256k1")
	}

	return publicKey.XBytes(), nil
}

func newFrostSigningOutput(message string, messageHash []byte, pubKey []byte, result *common.SignatureData) frostSigningOutput {
	signature := []byte(nil)
	if result != nil {
		signature = result.Signature
	}

	return frostSigningOutput{
		Protocol:    "frost",
		Curve:       "secp256k1",
		Message:     message,
		MessageHash: hexutil.Encode(messageHash),
		PubKey:      hexutil.Encode(pubKey),
		Signature:   hexutil.Encode(signature),
	}
}

func saveSigningResult(result frostSigningOutput) error {
	raw, err := json.Marshal(result)
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
	default:
		return errors.Errorf("unsupported output type: %s", utils.OutputType)
	}
	return nil
}
