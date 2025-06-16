package reshare

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/bitcoin"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	bitcoinResharing "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/bitcoin"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var consolidateParams = bitcoin.DefaultConsolidateOutputsParams

func init() {
	registerReshareBtcOptions(reshareBtcCmd)
}

func registerReshareBtcOptions(cmd *cobra.Command) {
	cmd.Flags().Uint64Var(&consolidateParams.FeeRate, "fee-rate", consolidateParams.FeeRate, "Fee rate for the transaction (sats/vbyte)")
	cmd.Flags().IntVar(&consolidateParams.OutputsCount, "outputs-count", consolidateParams.OutputsCount, "Number of outputs to split the funds into")
	cmd.Flags().IntVar(&consolidateParams.MaxInputsCount, "max-inputs-count", consolidateParams.MaxInputsCount, "Maximum number of inputs to use in the transaction")
}

var reshareBtcCmd = &cobra.Command{
	Use:   "bitcoin [chain-id] [target-addr]",
	Short: "Command for service migration during key resharing for Bitcoin",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := cfg.SecretsStorage()
		share, err := storage.GetTssShare()
		if err != nil {
			return errors.Wrap(err, "failed to get tss share")
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

		var client *bitcoin.Client
		for _, ch := range cfg.Chains() {
			if ch.Id == args[0] && ch.Type == chain.TypeBitcoin {
				client = bitcoin.NewBridgeClient(bitcoin.FromChain(ch))
				break
			}
		}
		if client == nil {
			return errors.New("bitcoin client configuration not found")
		}
		targetAddr, err := btcutil.DecodeAddress(args[1], client.ChainParams())
		if err != nil {
			return errors.Wrap(err, "failed to decode target address")
		}

		connectionManager := p2p.NewConnectionManager(
			parties,
			p2p.PartyStatus_PS_RESHARE,
			cfg.Log().WithField("component", "connection_manager"),
		)

		session := bitcoinResharing.NewSession(
			tss.LocalSignParty{
				Account:   *account,
				Share:     share,
				Threshold: cfg.TssSessionParams().Threshold,
			},
			client,
			bitcoinResharing.SessionParams{
				ConsolidateParams: consolidateParams,
				TargetAddr:        targetAddr,
				SessionParams:     cfg.TssSessionParams(),
			},
			parties,
			connectionManager.GetReadyCount,
			cfg.Log().WithField("component", "btc_reshare_session"),
		)

		sessionManager := p2p.NewSessionManager(session)

		errGroup := new(errgroup.Group)
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		errGroup.Go(func() error {
			server := p2p.NewServer(
				cfg.P2pGrpcListener(),
				sessionManager,
				parties,
				*cert,
				cfg.Log().WithField("component", "p2p_server"),
			)
			server.SetStatus(p2p.PartyStatus_PS_RESHARE)
			return server.Run(ctx)
		})

		errGroup.Go(func() error {
			defer cancel()

			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run bitcoin resharing session")
			}
			txHash, err := session.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain migration tx hash")
			}
			if txHash == "" {
				cfg.Log().Info("local party is not a part of the resharing session")
				return nil
			}

			cfg.Log().Info("bitcoin resharing session successfully completed")
			cfg.Log().Info(fmt.Sprintf("Migration transaction hash: %s", txHash))

			return nil
		})

		return errGroup.Wait()
	},
}
