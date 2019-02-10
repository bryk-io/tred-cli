package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var keyCmd = &cobra.Command{
	Use:     "key",
	Example: "tred key -s 128",
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
	size := 128
	b64 := false
	keyCmd.Flags().IntVarP(&size, "size", "s", size, "size (in bytes) for the key value")
	keyCmd.Flags().BoolVarP(&b64, "encode", "e", b64, "encode the key in base64")
	if err := viper.BindPFlag("key.size", keyCmd.Flags().Lookup("size")); err != nil {
		log.Fatal(err)
	}
	if err := viper.BindPFlag("key.encode", keyCmd.Flags().Lookup("encode")); err != nil {
		log.Fatal(err)
	}
	rootCmd.AddCommand(keyCmd)
}
