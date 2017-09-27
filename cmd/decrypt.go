package cmd

import (
  "bytes"
  "errors"
  "fmt"
  "github.com/bryk-io/x/crypto/tred"
  "github.com/spf13/cobra"
  "github.com/spf13/viper"
  "io/ioutil"
  "os"
  "path/filepath"
  "strings"
  "time"
)

var decryptCmd = &cobra.Command{
  Use:           "decrypt input",
  Aliases:       []string{"dec", "open"},
  Short:         "Decrypt provided file or directory",
  RunE:          runDecrypt,
  SilenceErrors: true,
  SilenceUsage:  true,
}

func init() {
  var (
    cipher string
    clean  bool
  )
  decryptCmd.Flags().StringVar(&cipher, "cipher", "aes", "cipher suite to use, 'aes' or 'chacha'")
  decryptCmd.Flags().BoolVar(&clean, "clean", false, "remove sealed files after decrypt")
  viper.BindPFlag("decrypt.cipher", decryptCmd.Flags().Lookup("cipher"))
  viper.BindPFlag("decrypt.clean", decryptCmd.Flags().Lookup("clean"))
  RootCmd.AddCommand(decryptCmd)
}

func decryptFile(w *tred.Worker, file string, clean bool) (*tred.Result, error) {
  fmt.Printf("Processing: %s\n", file)
  input, err := os.Open(file)
  if err != nil {
    return nil, err
  }
  defer input.Close()
  
  output, err := os.Create(strings.Replace(file, "_enc", "", 1))
  if err != nil {
    return nil, err
  }
  defer output.Close()
  
  // Remove original file is requested
  if clean {
    defer os.Remove(file)
  }
  
  return w.Decrypt(input, output)
}

func runDecrypt(_ *cobra.Command, args []string) error {
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
  switch viper.GetString("decrypt.cipher") {
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
  if ! bytes.Equal(key, confirmation) {
    return errors.New("provided keys don't match")
  }
  
  // Get worker instance
  conf := tred.DefaultConfig(key)
  conf.Cipher = cs
  w := tred.NewWorker(conf)
  fmt.Printf("\n")
  
  clean := viper.GetBool("decrypt.clean")
  var total time.Duration
  if info.IsDir() {
    // Process all files inside the input directory
    files, err := ioutil.ReadDir(path)
    if err != nil {
      return err
    }
    
    for _, file := range files {
      if ! file.IsDir() {
        res, err := decryptFile(w, filepath.Join(path, file.Name()), clean)
        if err != nil {
          return err
        }
        total += res.Duration
      }
    }
  } else {
    // Process single file
    res, err := decryptFile(w, path, clean)
    if err != nil {
      return err
    }
    total = res.Duration
  }
  
  fmt.Printf("Done in: %v\n", total)
  return nil
}