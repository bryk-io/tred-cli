package cmd

import (
  "fmt"
  "github.com/cheggaaa/pb"
  "github.com/spf13/cobra"
  "golang.org/x/crypto/ssh/terminal"
  "os"
  "path/filepath"
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

// Get a progress bar for based on a file details
func getProgressBar(info os.FileInfo) *pb.ProgressBar {
  prefix := fmt.Sprintf("%-30s", filepath.Base(info.Name()))
  bar := pb.New(int(info.Size())).SetUnits(pb.U_BYTES).Prefix(prefix)
  bar.SetWidth(100)
  bar.SetMaxWidth(100)
  return bar
}