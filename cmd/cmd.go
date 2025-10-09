package cmd

import (
	"github.com/spf13/cobra"
)

var (
	configPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "chaos-proxy",
	Short: "A proxy server for injecting configurable network chaos",
	Long: `Chaos Proxy is a proxy server for injecting configurable network chaos 
(latency, failures, connection drops, rate-limiting, etc.) into any HTTP or HTTPS traffic.`,
}

func init() {
	rootCmd.Flags().StringVar(&configPath, "config", "chaos.yaml", "YAML config file path")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Print loaded middlewares and request logs")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// GetConfigPath returns the config file path.
func GetConfigPath() string {
	return configPath
}

// GetVerbose returns whether verbose logging is enabled.
func GetVerbose() bool {
	return verbose
}
