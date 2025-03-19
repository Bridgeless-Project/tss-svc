package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tss "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var defaultGenerationDeadline = 10 * time.Minute

func init() {
	utils.RegisterOutputFlags(preparamsCmd)
}

var preparamsCmd = &cobra.Command{
	Use:   "preparams",
	Short: "Generates pre-parameters for the TSS protocol",
	Args:  cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !utils.OutputValid() {
			return errors.New("invalid output type")
		}

		return nil
	},
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

		return storePreParams(cmd, params)
	},
}

func storePreParams(cmd *cobra.Command, params *tss.LocalPreParams) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "failed to marshal pre-parameters")
	}

	switch utils.OutputType {
	case "console":
		fmt.Println(string(raw))
	case "file":
		fmt.Println(utils.FilePath)
		if err = os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write pre-parameters to file")
		}
	case "vault":
		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		if err := storage.SaveKeygenPreParams(params); err != nil {
			return errors.Wrap(err, "failed to save pre-parameters to vault")
		}
	}

	return nil
}
