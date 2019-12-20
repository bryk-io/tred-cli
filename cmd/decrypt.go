package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/x/cli"
)

var decryptCmd = &cobra.Command{
	Use:     "decrypt input",
	Aliases: []string{"dec", "open"},
	Example: "tred decrypt --cipher chacha -cra [INPUT]",
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
		{
			Name:      "all",
			Usage:     "include hidden files",
			FlagKey:   "decrypt.all",
			ByDefault: false,
			Short:     "a",
		},
		{
			Name:      "workers",
			Usage:     "number or workers to run for parallel processing",
			FlagKey:   "decrypt.workers",
			ByDefault: runtime.NumCPU(),
			Short:     "w",
		},
	}
	if err := cli.SetupCommandParams(decryptCmd, params); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(decryptCmd)
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

	// Start tred workers pool
	pool, err := newPool(viper.GetInt("decrypt.workers"), key, viper.GetString("decrypt.cipher"), log)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize TRED workers")
		return err
	}

	// Process input
	start := time.Now()
	if isDir(input) {
		_ = filepath.Walk(input, func(f string, i os.FileInfo, err error) error {
			// Unexpected error walking the directory
			if err != nil {
				log.WithFields(logrus.Fields{
					"location": f,
					"error":    err,
				}).Warn("failed to traverse location")
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") && !viper.GetBool("decrypt.all") {
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

			// Add job to processing pool
			pool.add(job{file: f, showBar: false})
			return nil
		})
	} else {
		// Process single file
		pool.add(job{file: input, showBar: true})
	}

	// Wait for operations to complete
	pool.done()
	log.WithFields(logrus.Fields{
		"time":  time.Since(start),
		"files": pool.count,
	}).Info("operation completed")
	return err
}
