// Package main is the entry point for the abrg CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/mbasa/abr-geocoder-go/internal/interface/cli"
)

func main() {
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
