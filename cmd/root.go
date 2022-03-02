// Package cmd provides a CLI tool to manage secure files at rest.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tredctl",
	Short: "CLI for the 'Tamper Resistant Encrypted Data' protocol",
}

// Execute adds all child commands to the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
