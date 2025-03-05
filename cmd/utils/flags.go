package utils

import (
	"github.com/hyle-team/tss-svc/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/distributed_lab/kit/kv"
)

const (
	configFlag = "config"
	syncFlag   = "sync"
)

func RegisterOutputFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&OutputType, "output", "o", "console", "Output type: console, file, or vault")
	cmd.Flags().StringVar(&FilePath, "path", "preparams.json", "Path to save the pre-parameters file (used when output-type is 'file')")
	RegisterConfigFlag(cmd)
}

func RegisterConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP(configFlag, "c", "config.yaml", "Path to the config file")
}

func RegisterSyncFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&SyncEnabled, "sync", "s", SyncEnabled, "Sync enabled/disabled (disabled default)")
}

func OutputValid() bool {
	return OutputType == "console" || OutputType == "file" || OutputType == "vault"
}

var OutputType string
var FilePath string
var SyncEnabled bool

func ConfigFromFlags(cmd *cobra.Command) (config.Config, error) {
	configPath, err := cmd.Flags().GetString(configFlag)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config flag")
	}

	// ensure that the viper is loaded
	viper := kv.NewViperFile(configPath)
	if _, err = viper.GetStringMap("ping"); err != nil {
		return nil, errors.Wrap(err, "failed to ping viper")
	}

	return config.New(viper), nil
}

func IsSyncEnabled(cmd *cobra.Command) bool {
	enabled, err := cmd.Flags().GetBool(syncFlag)
	if err != nil {
		return false
	}

	return enabled
}
