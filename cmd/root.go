package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"go.bryk.io/x/crypto/tred"
	"golang.org/x/crypto/ssh/terminal"
)

var rootCmd = &cobra.Command{
	Use:   "tred",
	Short: "CLI for the 'Tamper Resistant Encrypted Data' protocol",
}

// Execute adds all child commands to the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Helper method to securely read data from stdin
func secureAsk(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	return terminal.ReadPassword(0)
}

// Ask the user to enter a key phrase that will be used to expand a secure cryptographic key
func getInteractiveKey() ([]byte, error) {
	key, err := secureAsk("Encryption Key: ")
	if err != nil {
		return nil, err
	}
	confirmation, err := secureAsk("\nConfirm Key: \n")
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(key, confirmation) {
		return nil, errors.New("provided keys don't match")
	}
	return key, nil
}

// Get a progress bar for based on a file details
func getProgressBar(info os.FileInfo) *pb.ProgressBar {
	prefix := fmt.Sprintf("%-30s", filepath.Base(info.Name()))
	bar := pb.New(int(info.Size())).SetUnits(pb.U_BYTES).Prefix(prefix)
	bar.SetWidth(100)
	bar.SetMaxWidth(100)
	return bar
}

// Inspect if the passed in file path is a directory or not
func isDir(file string) bool {
	info, err := os.Stat(file)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Return a new TRED worker instance
func getWorker(key []byte, cipher string) (*tred.Worker, error) {
	// Get cipher suite
	var cs byte
	switch cipher {
	case "aes":
		cs = tred.AES
	case "chacha":
		cs = tred.CHACHA20
	default:
		return nil, errors.New("invalid cipher suite")
	}

	// Get worker instance
	conf, err := tred.DefaultConfig(key)
	if err != nil {
		return nil, err
	}
	conf.Cipher = cs
	return tred.NewWorker(conf)
}

func getLogger(silent bool) *logrus.Logger {
	ll := logrus.New()
	ll.Level = logrus.DebugLevel
	ll.Formatter = &prefixed.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.StampMilli,
	}
	if silent {
		ll.Out = ioutil.Discard
	}
	return ll
}
