package main

import (
	"fmt"
	"log"
	"os"

	"chaos-proxy-go/cmd"
	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/proxy"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	// Print version info on startup
	fmt.Printf("chaos-proxy-go version: %s (commit: %s, date: %s)\n", version, commit, date)

	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Parse flags
	configPath := cmd.GetConfigPath()
	verbose := cmd.GetVerbose()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if verbose {
		fmt.Printf("Loaded config from: %s\n", configPath)
		fmt.Printf("Target: %s\n", cfg.Target)
		fmt.Printf("Port: %d\n", cfg.Port)
	}

	// Start proxy server
	server := proxy.New(cfg, verbose)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
