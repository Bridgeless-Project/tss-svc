package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Bridgeless-Project/tss-svc/cmd/utils"
	rootTss "github.com/Bridgeless-Project/tss-svc/internal/tss"
	ecdsaTss "github.com/Bridgeless-Project/tss-svc/internal/tss/protocols/ecdsa"
	"github.com/bnb-chain/tss-lib/v3/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var defaultGenerationDeadline = 10 * time.Minute

func init() {
	utils.RegisterOutputFlags(preparamsCmd)
}

var preparamsCmd = &cobra.Command{
	Use:   "preparams [protocol]",
	Short: "Generates pre-parameters for the TSS protocol",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !utils.OutputValid() {
			return errors.New("invalid output type")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch rootTss.ProtocolType(args[0]) {
		case rootTss.ProtocolID_ECDSA:
			fmt.Println("Generating ECDSA pre-parameters...")

			params, err := keygen.GeneratePreParams(defaultGenerationDeadline)
			if err != nil {
				return errors.Wrap(err, "failed to generate pre-parameters")
			}
			if !params.ValidateWithProof() {
				return errors.New("generated pre-parameters are invalid, please try again")
			}

			fmt.Println("ECDSA pre-parameters generated successfully")

			pr := ecdsaTss.NewEcdsaPreParams()
			err = pr.SetData(params)
			if err != nil {
				return errors.Wrap(err, "failed to set preparams")
			}

			return storePreParams(cmd, pr)
		case rootTss.ProtocolID_FROST:

			// TODO: add preparams here
			fmt.Println("FROST keygen does not require pre-parameters")
			return nil
		default:
			return errors.Errorf("unsupported TSS protocol: %s", args[0])
		}
	},
}

func storePreParams(cmd *cobra.Command, params rootTss.PreParams) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "failed to marshal pre-parameters")
	}

	switch utils.OutputType {
	case "console":
		fmt.Println(string(raw))
	case "file":
		if err = os.WriteFile(utils.FilePath, raw, 0644); err != nil {
			return errors.Wrap(err, "failed to write pre-parameters to file")
		}
	case "vault":
		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		if err = storage.SaveKeygenPreParams(params); err != nil {
			return errors.Wrap(err, "failed to save pre-parameters to vault")
		}
	}

	return nil
}
