package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tss "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/hyle-team/tss-svc/internal/secrets/vault"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var defaultGenerationDeadline = 10 * time.Minute
var outputType string
var filePath string
var configPath string

func init() {
	registerPreParamsFlags(preparamsCmd)
}

var preparamsCmd = &cobra.Command{
	Use:   "preparams",
	Short: "Generates pre-parameters for the TSS protocol",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Generating pre-parameters...")

		params, err := tss.GeneratePreParams(defaultGenerationDeadline)
		if err != nil {
			return errors.Wrap(err, "failed to generate pre-parameters")
		}
		if !params.ValidateWithProof() {
			return errors.New("generated pre-parameters are invalid, please try again")
		}

		fmt.Println("Pre-parameters generated successfully")

		switch outputType {
		case "console":
			raw, _ := json.Marshal(params)
			fmt.Println(string(raw))
		case "file":
			raw, _ := json.Marshal(params)
			if err := os.WriteFile(filePath, raw, 0644); err != nil {
				return errors.Wrap(err, "failed to write pre-parameters to file")
			}
		case "vault":
			config, err := utils.ConfigFromFlags(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to get config from flags")
			}

			storage := vault.NewStorage(config.VaultClient())
			if err := storage.SaveKeygenPreParams(params); err != nil {
				return errors.Wrap(err, "failed to save pre-parameters to vault")
			}
		}

		return nil
	},
}

func registerPreParamsFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&outputType, "output", "o", "console", "Output type: console, file, or vault")
	cmd.Flags().StringVar(&filePath, "path", "preparams.json", "Path to save the pre-parameters file (used when output-type is 'file')")
	cmd.Flags().StringVar(&configPath, "config", "config.yaml", "Path to configuration file (used when output-type is 'vault')")
}
