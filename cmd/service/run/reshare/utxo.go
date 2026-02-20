package reshare

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	utxochain "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	utxoutils "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	utxoResharing "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/utxo"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	consolidateParams   = utxoutils.DefaultResharingParams
	maxFeeRateSatsPerKb = int64(consolidateParams.MaxFeeRateSatsPerKb)
)

func init() {
	registerReshareUtxoOptions(reshareUtxoCmd)
}

func registerReshareUtxoOptions(cmd *cobra.Command) {
	cmd.Flags().UintVar(&consolidateParams.SetParams[0].OutsCount, "outputs-count", consolidateParams.SetParams[0].OutsCount, "Number of outputs to split the funds into")
	cmd.Flags().UintVar(&consolidateParams.SetParams[0].MaxInputsCount, "max-inputs-count", consolidateParams.SetParams[0].MaxInputsCount, "Maximum number of inputs to use in the transaction")
	cmd.Flags().Int64Var(&maxFeeRateSatsPerKb, "max-fee-rate", maxFeeRateSatsPerKb, "Maximum fee rate in sats per KB for the migration transaction")
}

var reshareUtxoCmd = &cobra.Command{
	Use:   "utxo [chain-id] [target-addr]",
	Short: "Command for service migration during key resharing for utxo chains",
	Args:  cobra.ExactArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		consolidateParams.MaxFeeRateSatsPerKb = btcutil.Amount(maxFeeRateSatsPerKb)
	},
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

		var cli client.Client
		for _, ch := range cfg.Chains() {
			if ch.Id == args[0] && ch.Type == chain.TypeBitcoin {
				cli = client.NewBridgeClient(utxochain.FromChain(ch))
				break
			}
		}
		if cli == nil {
			return errors.New("utxo client configuration not found")
		}
		targetAddr := args[1]
		if !cli.UtxoHelper().AddressValid(targetAddr) {
			return errors.New(fmt.Sprintf("invalid target address: %s", targetAddr))
		}
		if err != nil {
			return errors.Wrap(err, "failed to decode target address")
		}

		session := utxoResharing.NewSession(
			tss.LocalSignParty{
				Account:   *account,
				Share:     share,
				Threshold: cfg.TssSessionParams().Threshold,
			},
			cli,
			utxoResharing.SessionParams{
				ConsolidateParams: consolidateParams,
				TargetAddr:        targetAddr,
				SessionParams:     cfg.TssSessionParams(),
			},
			parties,
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

			// FIXME: handle start time
			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run utxo resharing session")
			}
			txHash, err := session.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain migration tx hash")
			}
			if txHash == "" {
				cfg.Log().Info("local party is not a part of the resharing session")
				return nil
			}

			cfg.Log().Info("utxo resharing session successfully completed")
			cfg.Log().Info(fmt.Sprintf("Migration transaction hash: %s", txHash))

			return nil
		})

		return errGroup.Wait()
	},
}
