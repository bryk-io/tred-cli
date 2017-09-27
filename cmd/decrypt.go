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
  "text/tabwriter"
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
    suffix string
    clean  bool
    silent bool
  )
  decryptCmd.Flags().StringVar(&cipher, "cipher", "aes", "cipher suite to use, 'aes' or 'chacha'")
  decryptCmd.Flags().StringVar(&suffix, "suffix", "_enc", "suffix to remove from encrypted files")
  decryptCmd.Flags().BoolVar(&clean, "clean", false, "remove sealed files after decrypt")
  decryptCmd.Flags().BoolVar(&silent, "silent", false, "suppress all output")
  viper.BindPFlag("decrypt.cipher", decryptCmd.Flags().Lookup("cipher"))
  viper.BindPFlag("decrypt.clean", decryptCmd.Flags().Lookup("clean"))
  viper.BindPFlag("decrypt.silent", decryptCmd.Flags().Lookup("silent"))
  viper.BindPFlag("decrypt.suffix", decryptCmd.Flags().Lookup("suffix"))
  RootCmd.AddCommand(decryptCmd)
}

func decryptFile(w *tred.Worker, file string) (string, []byte, error) {
  input, err := os.Open(file)
  if err != nil {
    return "", nil, err
  }
  defer input.Close()
  
  output, err := os.Create(strings.Replace(file, viper.GetString("decrypt.suffix"), "", 1))
  if err != nil {
    return "", nil, err
  }
  defer output.Close()
  
  res, err := w.Decrypt(input, output)
  if err == nil {
    if viper.GetBool("decrypt.clean") {
      defer os.Remove(file)
    }
  } else {
    os.Remove(output.Name())
  }
  return filepath.Base(output.Name()), res.Checksum, err
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
  
  // Process input
  report := make(map[string]string)
  start := time.Now()
  if info.IsDir() {
    // Process all files inside the input directory
    files, err := ioutil.ReadDir(path)
    if err != nil {
      return err
    }
    
    for _, file := range files {
      if ! file.IsDir() && ! strings.HasPrefix(file.Name(), ".") {
        file, checksum, err := decryptFile(w, filepath.Join(path, file.Name()))
        if err != nil {
          return err
        }
        report[file] = fmt.Sprintf("%x", checksum)
      }
    }
  } else {
    // Process single file
    file, checksum, err := decryptFile(w, path)
    if err != nil {
      return err
    }
    report[file] = fmt.Sprintf("%x", checksum)
  }
  
  if ! viper.GetBool("decrypt.silent") {
    fmt.Printf("\n")
    output := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
    for k, v := range report {
      fmt.Fprintf(output, "%s\t%s\n", k, v)
    }
    output.Flush()
    fmt.Printf("=== Done in: %v\n", time.Since(start))
  }
  return nil
}
