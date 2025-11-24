package reshare

import (
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	zanoResharing "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/zano"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var isEthOwner bool

func init() {
	registerReshareZanoOptions(reshareZanoCmd)
}

func registerReshareZanoOptions(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&isEthOwner, "is-eth-key", false, "Indicates if the new owner public key is an Ethereum key")
}

var reshareZanoCmd = &cobra.Command{
	Use:   "zano [asset-id] [owner-pub-key]",
	Short: "Command for service migration during key resharing for Zano network",
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

		var client *zano.Client
		for _, ch := range cfg.Chains() {
			if ch.Type == chain.TypeZano {
				client = zano.NewBridgeClient(zano.FromChain(ch))
				break
			}
		}
		if client == nil {
			return errors.New("zano client configuration not found")
		}

		connectionManager := p2p.NewConnectionManager(
			parties,
			p2p.PartyStatus_PS_RESHARE,
			cfg.Log().WithField("component", "connection_manager"),
		)

		session := zanoResharing.NewSession(
			tss.LocalSignParty{
				Account:   *account,
				Share:     share,
				Threshold: cfg.TssSessionParams().Threshold,
			},
			client,
			zanoResharing.SessionParams{
				AssetId:       args[0],
				OwnerPubKey:   args[1],
				IsEthKey:      isEthOwner,
				SessionParams: cfg.TssSessionParams(),
			},
			parties,
			connectionManager.GetReadyCount,
			cfg.Log().WithField("component", "zano_reshare_session"),
		)

		sessionManager := p2p.NewSessionManager(session)

		eg := new(errgroup.Group)
		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		eg.Go(func() error {
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

		eg.Go(func() error {
			defer cancel()

			if err := session.Run(ctx); err != nil {
				return errors.Wrap(err, "failed to run zano resharing session")
			}
			txHash, err := session.WaitFor()
			if err != nil {
				return errors.Wrap(err, "failed to obtain migration tx hash")
			}
			if txHash == "" {
				cfg.Log().Info("local party is not an active part of the resharing session")
				return nil
			}

			cfg.Log().Infof("zano resharing session for asset %q successfully completed", args[0])
			cfg.Log().Info(fmt.Sprintf("Migration transaction hash: %s", txHash))

			return nil
		})

		return eg.Wait()
	},
}
