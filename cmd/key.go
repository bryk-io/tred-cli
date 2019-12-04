package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"

	"github.com/bryk-io/x/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keyCmd = &cobra.Command{
	Use:     "key",
	Example: "tred key --encode",
	Short:   "Generate a random and secure key value",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Generate
		k := make([]byte, viper.GetInt("key.size"))
		if _, err := rand.Reader.Read(k); err != nil {
			return err
		}

		// Encode
		if viper.GetBool("key.encode") {
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
		},
		{
			Name:      "encode",
			Usage:     "encode the key in base64",
			FlagKey:   "key.encode",
			ByDefault: false,
		},
	}
	if err := cli.SetupCommandParams(keyCmd, params); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(keyCmd)
}
