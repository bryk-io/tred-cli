package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set at compile time in the Makefile
var (
	buildCode  string
	releaseTag string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"info"},
	Short:   "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Home: https://github.com/bryk-io/tred-cli")
		fmt.Println("Release:", releaseTag)
		fmt.Println("Build Code:", buildCode)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
