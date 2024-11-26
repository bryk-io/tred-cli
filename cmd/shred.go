package cmd

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/pkg/cli"
	viperUtils "go.bryk.io/pkg/cli/viper"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
)

var shredCmd = &cobra.Command{
	Use:     "shred",
	Aliases: []string{"del", "rm"},
	Example: "tred shred -ra [INPUT]",
	Short:   "Securely delete files/directories while preventing contents recovery",
	RunE:    runShred,
}

func init() {
	params := []cli.Param{
		{
			Name:      "recursive",
			Usage:     "recursively process directories",
			FlagKey:   "shred.recursive",
			ByDefault: false,
			Short:     "r",
		},
		{
			Name:      "all",
			Usage:     "include hidden files",
			FlagKey:   "shred.all",
			ByDefault: false,
			Short:     "a",
		},
		{
			Name:      "workers",
			Usage:     "number or workers to run for parallel processing",
			FlagKey:   "shred.workers",
			ByDefault: runtime.NumCPU(),
			Short:     "w",
		},
		{
			Name:      "silent",
			Usage:     "suppress all output",
			FlagKey:   "shred.silent",
			ByDefault: false,
			Short:     "s",
		},
	}
	if err := cli.SetupCommandParams(shredCmd, params); err != nil {
		panic(err)
	}
	if err := viperUtils.BindFlags(shredCmd, params, viper.GetViper()); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(shredCmd)
}

func runShred(_ *cobra.Command, args []string) error {
	log := getLogger(viper.GetBool("shred.silent"))

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

	// Get temporary encryption key
	key := make([]byte, 64)
	if _, err := rand.Read(key); err != nil {
		return errors.Wrap(err, "generate encryption key")
	}

	// Start tred workers pool
	wp, err := newPool(viper.GetInt("shred.workers"), key, "chacha", log)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize TRED workers")
		return err
	}

	// Process input
	cleanUpDirs := []string{}
	start := time.Now()
	// nolint: nestif
	if isDir(input) {
		err = filepath.Walk(input, func(f string, i os.FileInfo, err error) error {
			// Unexpected error walking the directory
			if err != nil {
				log.WithFields(xlog.Fields{
					"location": f,
					"error":    err,
				}).Warning("failed to traverse location")
				return err
			}

			// Ignore hidden files
			if strings.HasPrefix(filepath.Base(f), ".") && !viper.GetBool("shred.all") {
				log.WithFields(xlog.Fields{
					"location": f,
				}).Debug("ignoring hidden file")
				return nil
			}

			// Don't go into subdirectories if not required
			if i.IsDir() && !viper.GetBool("shred.recursive") {
				log.WithFields(xlog.Fields{
					"location": f,
				}).Debug("ignoring directory on non-recursive run")
				return filepath.SkipDir
			}

			// Ignore subdir markers
			if i.IsDir() {
				cleanUpDirs = append(cleanUpDirs, f)
				return nil
			}

			// Add job to processing pool
			wp.add(job{file: f, showBar: false, shred: true})
			return nil
		})
		cleanUpDirs = append(cleanUpDirs, input)
	} else {
		// Process single file
		wp.add(job{file: input, showBar: true, shred: true})
	}

	// Wait for operations to complete
	wp.done()
	for _, td := range cleanUpDirs {
		_ = os.Remove(td) // remove empty directories, ignore errors
	}
	log.WithFields(xlog.Fields{
		"time":  time.Since(start),
		"files": wp.count,
	}).Info("operation completed")
	return err
}
