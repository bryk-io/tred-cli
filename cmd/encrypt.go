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

var encryptCmd = &cobra.Command{
	Use:     "encrypt",
	Aliases: []string{"enc", "seal"},
	Example: "tred encrypt --cipher chacha -cra [INPUT]",
	Short:   "Encrypt provided file or directory",
	RunE:    runEncrypt,
}

func init() {
	params := []cli.Param{
		{
			Name:      "cipher",
			Usage:     "cipher suite to use, 'aes' or 'chacha'",
			FlagKey:   "encrypt.cipher",
			ByDefault: "aes",
		},
		{
			Name:      "suffix",
			Usage:     "suffix to add on encrypted files",
			FlagKey:   "encrypt.suffix",
			ByDefault: ".tred",
		},
		{
			Name:      "key",
			Usage:     "load encryption key from an existing file, or interactively enter one",
			FlagKey:   "encrypt.key",
			ByDefault: "",
		},
		{
			Name:      "clean",
			Usage:     "remove original files after encrypt",
			FlagKey:   "encrypt.clean",
			ByDefault: false,
			Short:     "c",
		},
		{
			Name:      "silent",
			Usage:     "suppress all output",
			FlagKey:   "encrypt.silent",
			ByDefault: false,
			Short:     "s",
		},
		{
			Name:      "recursive",
			Usage:     "recursively process directories",
			FlagKey:   "encrypt.recursive",
			ByDefault: false,
			Short:     "r",
		},
		{
			Name:      "all",
			Usage:     "include hidden files",
			FlagKey:   "encrypt.all",
			ByDefault: false,
			Short:     "a",
		},
		{
			Name:      "workers",
			Usage:     "number or workers to run for parallel processing",
			FlagKey:   "encrypt.workers",
			ByDefault: runtime.NumCPU(),
			Short:     "w",
		},
	}
	if err := cli.SetupCommandParams(encryptCmd, params); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(encryptCmd)
}

func runEncrypt(_ *cobra.Command, args []string) error {
	log := getLogger(viper.GetBool("encrypt.silent"))

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

	// Get encryption key
	var key []byte
	if viper.GetString("encrypt.key") != "" {
		if key, err = ioutil.ReadFile(viper.GetString("encrypt.key")); err != nil {
			log.WithField("error", err).Fatal("could not read key file provided")
			return err
		}
	} else {
		log.Info("no key file provided, asking for a secret key now")
		if key, err = getInteractiveKey(); err != nil {
			log.WithField("error", err).Fatal("failed to retrieve key")
			return err
		}
	}

	// Start tred workers pool
	pool, err := newPool(viper.GetInt("encrypt.workers"), key, viper.GetString("encrypt.cipher"), log)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize TRED workers")
		return err
	}

	// Process input
	start := time.Now()
	if isDir(input) {
		err = filepath.Walk(input, func(f string, i os.FileInfo, err error) error {
			// Unexpected error walking the directory
			if err != nil {
				log.WithFields(logrus.Fields{
					"location": f,
					"error":    err,
				}).Warn("failed to traverse location")
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") && !viper.GetBool("encrypt.all") {
				log.WithFields(logrus.Fields{
					"location": f,
				}).Debug("ignoring hidden file")
				return nil
			}

			// Don't go into sub-directories if not required
			if i.IsDir() && !viper.GetBool("encrypt.recursive") {
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
			pool.add(job{file: f, showBar: false, encrypt: true})
			return nil
		})
	} else {
		// Process single file
		pool.add(job{file: input, showBar: true, encrypt: true})
	}

	// Wait for operations to complete
	pool.done()
	log.WithFields(logrus.Fields{
		"time":  time.Since(start),
		"files": pool.count,
	}).Info("operation completed")
	return err
}
