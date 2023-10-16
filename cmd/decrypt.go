package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/pkg/cli"
	xlog "go.bryk.io/pkg/log"
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
			FlagKey:   "decrypt.cipher",
			ByDefault: "aes",
		},
		{
			Name:      "suffix",
			Usage:     "suffix to remove from encrypted files",
			FlagKey:   "decrypt.suffix",
			ByDefault: ".tred",
		},
		{
			Name:      "key",
			Usage:     "load decryption key from an existing file",
			FlagKey:   "decrypt.key",
			ByDefault: "",
		},
		{
			Name:      "clean",
			Usage:     "remove sealed files after decryption",
			FlagKey:   "decrypt.clean",
			ByDefault: false,
			Short:     "c",
		},
		{
			Name:      "silent",
			Usage:     "suppress all output",
			FlagKey:   "decrypt.silent",
			ByDefault: false,
			Short:     "s",
		},
		{
			Name:      "recursive",
			Usage:     "recursively process directories",
			FlagKey:   "decrypt.recursive",
			ByDefault: false,
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
	if err := cli.SetupCommandParams(decryptCmd, params, viper.GetViper()); err != nil {
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

	// ... from file
	if viper.GetString("decrypt.key") != "" {
		if key, err = os.ReadFile(viper.GetString("decrypt.key")); err != nil {
			log.WithField("error", err).Fatal("could not read key file provided")
			return err
		}
	}

	// ... from user input
	if len(key) == 0 {
		log.Info("no key file provided, asking for a secret key now")
		if key, err = secureAsk("Decryption Key: \n"); err != nil {
			log.WithField("error", err).Fatal("failed to retrieve key")
			return err
		}
	}

	// Start TRED worker pool
	wp, err := newPool(viper.GetInt("decrypt.workers"), key, viper.GetString("decrypt.cipher"), log)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize TRED workers")
		return err
	}

	// Process input
	start := time.Now()
	if isDir(input) {
		_ = filepath.Walk(input, func(f string, i os.FileInfo, err error) error {
			fields := xlog.Fields{"location": f}

			// Unexpected error walking the directory
			if err != nil {
				fields["error"] = err
				log.WithFields(fields).Warning("failed to traverse location")
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") && !viper.GetBool("decrypt.all") {
				log.WithFields(fields).Debug("ignoring hidden file")
				return nil
			}

			// Don't go into sub-directories if not required
			if i.IsDir() && !viper.GetBool("decrypt.recursive") {
				log.WithFields(fields).Debug("ignoring directory on non-recursive run")
				return filepath.SkipDir
			}

			// Add files as jobs on processing pool
			if !i.IsDir() {
				wp.add(job{file: f, showBar: false})
			}
			return nil
		})
	} else {
		// Process single file
		wp.add(job{file: input, showBar: true})
	}

	// Wait for operations to complete
	wp.done()
	log.WithFields(xlog.Fields{
		"time":  time.Since(start),
		"files": wp.count,
	}).Info("operation completed")
	return err
}
