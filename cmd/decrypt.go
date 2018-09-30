package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bryk-io/x/crypto/tred"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var decryptCmd = &cobra.Command{
	Use:           "decrypt input",
	Aliases:       []string{"dec", "open"},
	Example:       "tred decrypt -dr -c chacha [INPUT]",
	Short:         "Decrypt provided file or directory",
	RunE:          runDecrypt,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	var (
		err       error
		cipher    string
		suffix    string
		clean     bool
		silent    bool
		recursive bool
	)
	decryptCmd.Flags().StringVarP(&cipher, "cipher", "c", "aes", "cipher suite to use, 'aes' or 'chacha'")
	decryptCmd.Flags().StringVar(&suffix, "suffix", "_enc", "suffix to remove from encrypted files")
	decryptCmd.Flags().BoolVarP(&clean, "clean", "d", false, "remove sealed files after decrypt")
	decryptCmd.Flags().BoolVarP(&silent, "silent", "s", false, "suppress all output")
	decryptCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "recursively process directories")
	err = viper.BindPFlag("decrypt.cipher", decryptCmd.Flags().Lookup("cipher"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("decrypt.clean", decryptCmd.Flags().Lookup("clean"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("decrypt.silent", decryptCmd.Flags().Lookup("silent"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("decrypt.suffix", decryptCmd.Flags().Lookup("suffix"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("decrypt.recursive", decryptCmd.Flags().Lookup("recursive"))
	if err != nil {
		log.Fatal(err)
	}
	rootCmd.AddCommand(decryptCmd)
}

func decryptFile(w *tred.Worker, file string, withBar bool) error {
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(strings.Replace(file, viper.GetString("decrypt.suffix"), "", 1))
	if err != nil {
		return err
	}
	defer output.Close()

	var r io.Reader
	r = input
	if !viper.GetBool("encrypt.silent") && withBar {
		info, err := input.Stat()
		if err != nil {
			return err
		}
		bar := getProgressBar(info)
		bar.Start()
		defer bar.Finish()
		r = bar.NewProxyReader(input)
	}
	_, err = w.Decrypt(r, output)
	if err == nil {
		if viper.GetBool("decrypt.clean") {
			defer os.Remove(file)
		}
	} else {
		defer os.Remove(output.Name())
		return err
	}
	return err
}

func runDecrypt(_ *cobra.Command, args []string) error {
	// Get input
	if len(args) == 0 {
		return errors.New("missing required input")
	}

	// Get input absolute path
	input, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	// Get encryption key
	key, err := secureAsk("\nEncryption Key: ")
	if err != nil {
		return err
	}

	// Get worker instance
	w, err := getWorker(key, viper.GetString("decrypt.cipher"))
	if err != nil {
		return err
	}

	// Process input
	if isDir(input) {
		wg := sync.WaitGroup{}
		err := filepath.Walk(input, func(f string, i os.FileInfo, err error) error {
			// Unexpected error walking the directory
			if err != nil {
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") {
				return nil
			}

			// Don't go into sub-directories if not required
			if i.IsDir() && !viper.GetBool("decrypt.recursive") {
				return filepath.SkipDir
			}

			// Ignore subdir markers
			if i.IsDir() {
				return nil
			}

			// Process regular files
			wg.Add(1)
			go func(entry string) {
				if err := decryptFile(w, entry, false); err != nil {
					fmt.Printf("ERROR: %s for: %s\n", err, entry)
				}
				if !viper.GetBool("decrypt.silent") {
					fmt.Printf(">> %s\n", entry)
				}
				wg.Done()
			}(f)
			return nil
		})
		wg.Wait()
		return err
	}

	// Process single file
	return decryptFile(w, input, true)
}
