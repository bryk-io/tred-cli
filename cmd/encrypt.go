package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bryk-io/x/crypto/tred"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var encryptCmd = &cobra.Command{
	Use:           "encrypt input",
	Aliases:       []string{"enc", "seal"},
	Short:         "Encrypt provided file or directory",
	RunE:          runEncrypt,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	var (
		cipher string
		suffix string
		report string
		clean  bool
		silent bool
	)
	encryptCmd.Flags().StringVar(&cipher, "cipher", "aes", "cipher suite to use, 'aes' or 'chacha'")
	encryptCmd.Flags().StringVar(&suffix, "suffix", "_enc", "suffix to add on encrypted files")
	encryptCmd.Flags().StringVar(&report, "report", "", "generate a JSON report of the process")
	encryptCmd.Flags().BoolVar(&clean, "clean", false, "remove original files after encrypt")
	encryptCmd.Flags().BoolVar(&silent, "silent", false, "suppress all output")
	viper.BindPFlag("encrypt.cipher", encryptCmd.Flags().Lookup("cipher"))
	viper.BindPFlag("encrypt.clean", encryptCmd.Flags().Lookup("clean"))
	viper.BindPFlag("encrypt.report", encryptCmd.Flags().Lookup("report"))
	viper.BindPFlag("encrypt.silent", encryptCmd.Flags().Lookup("silent"))
	viper.BindPFlag("encrypt.suffix", encryptCmd.Flags().Lookup("suffix"))
	rootCmd.AddCommand(encryptCmd)
}

func encryptFile(w *tred.Worker, file string, withBar bool) (string, []byte, error) {
	input, err := os.Open(file)
	if err != nil {
		return "", nil, err
	}
	defer input.Close()

	output, err := os.Create(fmt.Sprintf("%s%s", file, viper.GetString("encrypt.suffix")))
	if err != nil {
		return "", nil, err
	}
	defer output.Close()

	var r io.Reader
	r = input
	if !viper.GetBool("encrypt.silent") && withBar {
		info, _ := input.Stat()
		bar := getProgressBar(info)
		bar.Start()
		defer bar.Finish()
		r = bar.NewProxyReader(input)
	}
	res, err := w.Encrypt(r, output)
	if err == nil {
		if viper.GetBool("encrypt.clean") {
			defer os.Remove(file)
		}
	}
	return filepath.Base(input.Name()), res.Checksum, nil
}

func runEncrypt(_ *cobra.Command, args []string) error {
	// Get input
	if len(args) == 0 {
		return errors.New("missing required input")
	}

	// Get input absolute path
	path, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Get cipher suite
	var cs byte
	switch viper.GetString("encrypt.cipher") {
	case "aes":
		cs = tred.AES_GCM
	case "chacha":
		cs = tred.CHACHA20
	default:
		return errors.New("invalid cipher suite")
	}

	// Get encryption key
	key, err := secureAsk("\nEncryption Key: ")
	if err != nil {
		return err
	}
	confirmation, err := secureAsk("\nConfirm Key: ")
	if err != nil {
		return err
	}
	if !bytes.Equal(key, confirmation) {
		return errors.New("provided keys don't match")
	}

	// Get worker instance
	conf := tred.DefaultConfig(key)
	conf.Cipher = cs
	w, err := tred.NewWorker(conf)
	if err != nil {
		return err
	}
	fmt.Printf("\n")

	// Process input
	report := make(map[string]string)
	start := time.Now()
	if info.IsDir() {
		// Process all files inside the input directory
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		wg := sync.WaitGroup{}
		for _, file := range files {
			if !file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
				wg.Add(1)
				go func(file os.FileInfo, report map[string]string) {
					f, checksum, err := encryptFile(w, filepath.Join(path, file.Name()), false)
					if err == nil {
						if !viper.GetBool("decrypt.silent") {
							fmt.Printf(">> %s\n", f)
						}
						report[f] = fmt.Sprintf("%x", checksum)
					}
					wg.Done()
				}(file, report)
			}
		}
		wg.Wait()
	} else {
		// Process single file
		file, checksum, err := encryptFile(w, path, true)
		if err != nil {
			return err
		}
		report[file] = fmt.Sprintf("%x", checksum)
	}

	if !viper.GetBool("encrypt.silent") {
		fmt.Printf("=== Done in: %v\n", time.Since(start))
	}

	// Generate JSON report
	if viper.GetString("encrypt.report") != "" {
		rp, err := filepath.Abs(viper.GetString("encrypt.report"))
		if err != nil {
			return err
		}
		rf, err := os.Create(rp)
		if err != nil {
			return err
		}
		js, _ := json.MarshalIndent(report, "", " ")
		rf.Write(js)
		rf.Close()
	}
	return nil
}
