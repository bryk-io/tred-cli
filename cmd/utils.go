package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"go.bryk.io/pkg/crypto/tred"
	xlog "go.bryk.io/pkg/log"
	"golang.org/x/term"
)

// Helper method to securely read data from stdin.
func secureAsk(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	return term.ReadPassword(0)
}

// Ask the user to enter a key phrase that will be used to
// expand a secure cryptographic key.
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

// Get a progress bar for based on a file details.
func getProgressBar(file *os.File) *pb.ProgressBar {
	info, err := file.Stat()
	if err != nil {
		return nil
	}
	bar := pb.New(int(info.Size()))
	bar.SetWidth(100)
	bar.SetMaxWidth(100)
	return bar
}

// Inspect if the passed in file path is a directory or not.
func isDir(file string) bool {
	info, err := os.Stat(file)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Return a new logging agent.
func getLogger(silent bool) xlog.Logger {
	if silent {
		return xlog.Discard()
	}
	return xlog.WithZero(xlog.ZeroOptions{
		PrettyPrint: true,
		ErrorField:  "error",
	})
}

// Return a new TRED worker instance.
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
