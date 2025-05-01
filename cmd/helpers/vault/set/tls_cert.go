package set

import (
	"crypto/tls"
	"os"

	"github.com/hyle-team/tss-svc/cmd/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var tlsCertCmd = &cobra.Command{
	Use:  "tls-cert [path-to-tls-cert] [path-to-tls-key]",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		certPath := args[0]
		keyPath := args[1]

		rawCert, err := os.ReadFile(certPath)
		if err != nil {
			return errors.Wrap(err, "failed to read TLS cert file")
		}

		rawKey, err := os.ReadFile(keyPath)
		if err != nil {
			return errors.Wrap(err, "failed to read TLS key file")
		}

		if _, err = tls.X509KeyPair(rawCert, rawKey); err != nil {
			return errors.Wrap(err, "failed to parse TLS cert and key")
		}

		config, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		storage := config.SecretsStorage()
		if err := storage.SaveLocalPartyTlsCertificate(rawCert, rawKey); err != nil {
			return errors.Wrap(err, "failed to save TLS certificate to vault")
		}

		return nil
	},
}
