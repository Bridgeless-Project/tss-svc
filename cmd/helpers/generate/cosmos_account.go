package generate

import (
	"encoding/json"
	"fmt"
	"os"

	secp256k1 "github.com/Bridgeless-Project/bridgeless-core/v12/crypto/ethsecp256k1"
	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	utils.RegisterOutputFlags(cosmosAccountCmd)
	registerCosmosAccountFlags(cosmosAccountCmd)
}

func registerCosmosAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&hrp, "hrp", "bridge", "Bech32 human-readable part of address")
}

var hrp string

var cosmosAccountCmd = &cobra.Command{
	Use: "cosmos-account",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !utils.OutputValid() {
			return errors.New("invalid output type")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		priv, err := secp256k1.GenerateKey()
		if err != nil {
			return errors.Wrap(err, "failed to generate private key")
		}

		account, err := core.NewAccount(hexutil.Encode(priv.Bytes()), hrp)
		if err != nil {
			return errors.Wrap(err, "failed to create account")
		}

		return storeAccount(cmd, account)
	},
}

func storeAccount(cmd *cobra.Command, account *core.Account) error {
	prvRaw := hexutil.Encode(account.PrivateKey().Bytes())[2:]

	switch utils.OutputType {
	case "console":
		fmt.Println("Private key:", prvRaw)
		fmt.Println("Address:", account.CosmosAddress().String())
	case "file":
		raw, _ := json.Marshal(map[string]string{
			"private_key": prvRaw,
			"address":     account.CosmosAddress().String(),
		})
		if err := os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write pre-parameters to file")
		}
	case "vault":
		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		if err = storage.SaveCoreAccount(account); err != nil {
			return errors.Wrap(err, "failed to save account to vault")
		}
	}

	return nil
}
