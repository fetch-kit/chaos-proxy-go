package main

import (
	"fmt"
	"os"

	"chaos-proxy-go/cmd"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	fmt.Printf("chaos-proxy-go version: %s (commit: %s, date: %s)\n", version, commit, date)

	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
