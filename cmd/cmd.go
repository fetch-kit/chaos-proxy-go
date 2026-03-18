package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/proxy"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if verbose {
			fmt.Printf("Loaded config from: %s\n", configPath)
			fmt.Printf("Target: %s\n", cfg.Target)
			fmt.Printf("Port: %d\n", cfg.Port)
		}
		server, err := proxy.New(cfg, verbose)
		if err != nil {
			return fmt.Errorf("failed to initialize proxy server: %w", err)
		}

		// Handle graceful shutdown on SIGINT/SIGTERM
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-quit
			log.Println("Shutting down server...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}
		}()

		if err := server.Start(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("failed to start server: %w", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&configPath, "config", "chaos.yaml", "YAML config file path")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Print loaded middlewares and request logs")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
