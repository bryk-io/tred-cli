package cmd

import (
  "fmt"
  "github.com/spf13/cobra"
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