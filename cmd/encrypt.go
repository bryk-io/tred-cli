package cmd

import (
	"fmt"
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
			ByDefault: "_enc",
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
	}
	if err := cli.SetupCommandParams(encryptCmd, params); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(encryptCmd)
}

func encryptFile(w *tred.Worker, file string, withBar bool) error {
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer func() {
		_ = input.Close()
	}()

	output, err := os.Create(fmt.Sprintf("%s%s", file, viper.GetString("encrypt.suffix")))
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
	_, err = w.Encrypt(r, output)
	if err == nil {
		if viper.GetBool("encrypt.clean") {
			defer func() {
				_ = os.Remove(file)
			}()
		}
	}
	return err
}

func runEncrypt(_ *cobra.Command, args []string) error {
	log := getLogger(viper.GetBool("encrypt.silent"))

	// Get input
	if len(args) == 0 {
		log.Fatal("missing required input")
		return errors.New("missing required input")
	}

	// Get input absolute path
	source, err := filepath.Abs(args[0])
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

	// Get worker instance
	w, err := getWorker(key, viper.GetString("encrypt.cipher"))
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize TRED worker")
		return err
	}

	// Process input
	start := time.Now()
	if isDir(source) {
		wg := sync.WaitGroup{}
		err := filepath.Walk(source, func(f string, i os.FileInfo, err error) error {
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

			// Process regular file
			wg.Add(1)
			go func(entry string, i os.FileInfo) {
				err := encryptFile(w, entry, false)
				if err != nil {
					log.WithFields(logrus.Fields{
						"file":  i.Name(),
						"error": err,
					}).Error("failed to encrypt file")
				} else {
					log.WithFields(logrus.Fields{
						"file": i.Name(),
					}).Debug("file encrypted")
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
	err = encryptFile(w, source, true)
	log.WithFields(logrus.Fields{"time": time.Since(start)}).Info("operation completed")
	return err
}
