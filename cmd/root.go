package cmd

import (
  "fmt"
  "github.com/spf13/cobra"
  "golang.org/x/crypto/ssh/terminal"
  "os"
)

var RootCmd = &cobra.Command{
  Use:   "tred",
  Short: "CLI for the 'Tamper Resistant Encrypted Data' protocol",
}

// Execute adds all child commands to the root command
func Execute() {
  if err := RootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

// Helper method to securely read data from stdin
func secureAsk(prompt string) ([]byte, error) {
  fmt.Print(prompt)
  return terminal.ReadPassword(0)
}