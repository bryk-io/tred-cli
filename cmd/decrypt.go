package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/x/cli"
	"go.bryk.io/x/crypto/tred"
)

var decryptCmd = &cobra.Command{
	Use:     "decrypt input",
	Aliases: []string{"dec", "open"},
	Example: "tred decrypt --cipher chacha -cr [INPUT]",
	Short:   "Decrypt provided file or directory",
	RunE:    runDecrypt,
}

func init() {
	params := []cli.Param{
		{
			Name:      "cipher",
			Usage:     "cipher suite to use, 'aes' or 'chacha'",
			ByDefault: "aes",
			FlagKey:   "decrypt.cipher",
		},
		{
			Name:      "suffix",
			Usage:     "suffix to remove from encrypted files",
			ByDefault: "_enc",
			FlagKey:   "decrypt.suffix",
		},
		{
			Name:      "key",
			Usage:     "load decryption key from an existing file",
			ByDefault: "",
			FlagKey:   "decrypt.key",
		},
		{
			Name:      "clean",
			Usage:     "remove sealed files after decryption",
			ByDefault: false,
			FlagKey:   "decrypt.clean",
			Short:     "c",
		},
		{
			Name:      "silent",
			Usage:     "suppress all output",
			ByDefault: false,
			FlagKey:   "decrypt.silent",
			Short:     "s",
		},
		{
			Name:      "recursive",
			Usage:     "recursively process directories",
			ByDefault: false,
			FlagKey:   "decrypt.recursive",
			Short:     "r",
		},
	}
	if err := cli.SetupCommandParams(decryptCmd, params); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(decryptCmd)
}

func decryptFile(w *tred.Worker, file string, withBar bool) error {
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer func() {
		_ = input.Close()
	}()

	output, err := os.Create(strings.Replace(file, viper.GetString("decrypt.suffix"), "", 1))
	if err != nil {
		return err
	}
	defer func() {
		_ = output.Close()
	}()

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
			defer func() {
				_ = os.Remove(file)
			}()
		}
	} else {
		defer func() {
			_ = os.Remove(output.Name())
		}()
		return err
	}
	return err
}

func runDecrypt(_ *cobra.Command, args []string) error {
	log := getLogger(viper.GetBool("decrypt.silent"))

	// Get input
	if len(args) == 0 {
		log.Fatal("missing required input")
		return errors.New("missing required input")
	}

	// Get input absolute path
	input, err := filepath.Abs(args[0])
	if err != nil {
		log.WithField("error", err).Fatal("invalid source provided")
		return err
	}

	// Get decryption key
	var key []byte
	if viper.GetString("decrypt.key") != "" {
		if key, err = ioutil.ReadFile(viper.GetString("decrypt.key")); err != nil {
			log.WithField("error", err).Fatal("could not read key file provided")
			return err
		}
	} else {
		log.Info("no key file provided, asking for a secret key now")
		if key, err = secureAsk("Decryption Key: \n"); err != nil {
			log.WithField("error", err).Fatal("failed to retrieve key")
			return err
		}
	}

	// Get worker instance
	w, err := getWorker(key, viper.GetString("decrypt.cipher"))
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize TRED worker")
		return err
	}

	// Process input
	start := time.Now()
	if isDir(input) {
		wg := sync.WaitGroup{}
		err := filepath.Walk(input, func(f string, i os.FileInfo, err error) error {
			// Unexpected error walking the directory
			if err != nil {
				log.WithFields(logrus.Fields{
					"location": f,
					"error":    err,
				}).Warn("failed to traverse location")
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") {
				log.WithFields(logrus.Fields{
					"location": f,
				}).Debug("ignoring hidden file")
				return nil
			}

			// Don't go into sub-directories if not required
			if i.IsDir() && !viper.GetBool("decrypt.recursive") {
				log.WithFields(logrus.Fields{
					"location": f,
				}).Debug("ignoring directory on non-recursive run")
				return filepath.SkipDir
			}

			// Ignore subdir markers
			if i.IsDir() {
				return nil
			}

			// Process regular files
			wg.Add(1)
			go func(entry string, i os.FileInfo) {
				if err := decryptFile(w, entry, false); err != nil {
					log.WithFields(logrus.Fields{
						"file":  i.Name(),
						"error": err,
					}).Error("failed to decrypt file")
				} else {
					log.WithFields(logrus.Fields{
						"file": i.Name(),
					}).Debug("file decrypted")
				}
				wg.Done()
			}(f, i)
			return nil
		})
		wg.Wait()
		log.WithFields(logrus.Fields{"time": time.Since(start)}).Info("operation completed")
		return err
	}

	// Process single file
	err = decryptFile(w, input, true)
	log.WithFields(logrus.Fields{"time": time.Since(start)}).Info("operation completed")
	return err
}
