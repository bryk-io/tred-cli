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

var encryptCmd = &cobra.Command{
	Use:           "encrypt",
	Aliases:       []string{"enc", "seal"},
	Example:       "tred encrypt -dr -c chacha [INPUT]",
	Short:         "Encrypt provided file or directory",
	RunE:          runEncrypt,
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
	encryptCmd.Flags().StringVarP(&cipher, "cipher", "c", "aes", "cipher suite to use, 'aes' or 'chacha'")
	encryptCmd.Flags().StringVar(&suffix, "suffix", "_enc", "suffix to add on encrypted files")
	encryptCmd.Flags().BoolVarP(&clean, "clean", "d", false, "remove original files after encrypt")
	encryptCmd.Flags().BoolVarP(&silent, "silent", "s", false, "suppress all output")
	encryptCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "recursively process directories")
	err = viper.BindPFlag("encrypt.cipher", encryptCmd.Flags().Lookup("cipher"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("encrypt.clean", encryptCmd.Flags().Lookup("clean"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("encrypt.silent", encryptCmd.Flags().Lookup("silent"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("encrypt.suffix", encryptCmd.Flags().Lookup("suffix"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("encrypt.recursive", encryptCmd.Flags().Lookup("recursive"))
	if err != nil {
		log.Fatal(err)
	}
	rootCmd.AddCommand(encryptCmd)
}

func encryptFile(w *tred.Worker, file string, withBar bool) error {
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(fmt.Sprintf("%s%s", file, viper.GetString("encrypt.suffix")))
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
	_, err = w.Encrypt(r, output)
	if err == nil {
		if viper.GetBool("encrypt.clean") {
			defer os.Remove(file)
		}
	}
	return err
}

func runEncrypt(_ *cobra.Command, args []string) error {
	// Get input
	if len(args) == 0 {
		return errors.New("missing required input")
	}

	// Get input absolute path
	source, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	// Get encryption key
	key, err := getInteractiveKey()
	if err != nil {
		return err
	}

	// Get worker instance
	w, err := getWorker(key, viper.GetString("encrypt.cipher"))
	if err != nil {
		return err
	}

	// Process input
	if isDir(source) {
		wg := sync.WaitGroup{}
		err := filepath.Walk(source, func(f string, i os.FileInfo, err error) error {
			// Unexpected error walking the directory
			if err != nil {
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") {
				return nil
			}

			// Don't go into sub-directories if not required
			if i.IsDir() && !viper.GetBool("encrypt.recursive") {
				return filepath.SkipDir
			}

			// Ignore subdir markers
			if i.IsDir() {
				return nil
			}

			// Process regular file
			wg.Add(1)
			go func(entry string) {
				err := encryptFile(w, entry, false)
				if err == nil && !viper.GetBool("encrypt.silent") {
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
	return encryptFile(w, source, true)
}
