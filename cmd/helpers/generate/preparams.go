package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tss "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var defaultGenerationDeadline = 10 * time.Minute

var preparamsCmd = &cobra.Command{
	Use:   "preparams [output-file.json]",
	Short: "Generates pre-parameters for the TSS protocol",
	Long:  "Generates pre-parameters for the TSS protocol. The default output is a configured stdout. Additionally, JSON file path can be specified to save data.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Generating pre-parameters...")

		params, err := tss.GeneratePreParams(defaultGenerationDeadline)
		if err != nil {
			return errors.Wrap(err, "failed to generate pre-parameters")
		}
		if !params.ValidateWithProof() {
			return errors.New("generated pre-parameters are invalid, please try again")
		}

		data, err := json.Marshal(params)
		if err != nil {
			return errors.Wrap(err, "failed to marshal pre-parameters")
		}

		fmt.Println("Generated pre-parameters:")
		fmt.Println(string(data))

		if len(args) == 0 {
			return nil
		}

		return errors.Wrap(os.WriteFile(args[0], data, 0644), "failed to write pre-parameters to file")
	},
}
