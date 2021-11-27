package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/pkg/cli"
)

var keyCmd = &cobra.Command{
	Use:     "key",
	Example: "tred key -es 512",
	Short:   "Generate a random and secure key value",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Generate
		k := make([]byte, viper.GetInt("key.size"))
		if _, err := rand.Reader.Read(k); err != nil {
			return err
		}

		// Encode
		if viper.GetBool("key.base64") {
			k = []byte(base64.StdEncoding.EncodeToString(k))
		}

		// Output
		if _, err := io.Copy(os.Stdout, bytes.NewReader(k)); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	params := []cli.Param{
		{
			Name:      "size",
			Usage:     "size (in bytes) for the key value",
			FlagKey:   "key.size",
			ByDefault: 128,
			Short:     "s",
		},
		{
			Name:      "base64",
			Usage:     "encode the key in base64",
			FlagKey:   "key.base64",
			ByDefault: false,
			Short:     "e",
		},
	}
	if err := cli.SetupCommandParams(keyCmd, params); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(keyCmd)
}
