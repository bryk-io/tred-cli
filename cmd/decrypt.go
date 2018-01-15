package cmd

import (
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
    report string
    clean  bool
    silent bool
  )
  decryptCmd.Flags().StringVar(&cipher, "cipher", "aes", "cipher suite to use, 'aes' or 'chacha'")
  decryptCmd.Flags().StringVar(&suffix, "suffix", "_enc", "suffix to remove from encrypted files")
  decryptCmd.Flags().StringVar(&report, "report", "", "validate input against a JSON report")
  decryptCmd.Flags().BoolVar(&clean, "clean", false, "remove sealed files after decrypt")
  decryptCmd.Flags().BoolVar(&silent, "silent", false, "suppress all output")
  viper.BindPFlag("decrypt.cipher", decryptCmd.Flags().Lookup("cipher"))
  viper.BindPFlag("decrypt.clean", decryptCmd.Flags().Lookup("clean"))
  viper.BindPFlag("decrypt.silent", decryptCmd.Flags().Lookup("silent"))
  viper.BindPFlag("decrypt.suffix", decryptCmd.Flags().Lookup("suffix"))
  viper.BindPFlag("decrypt.report", decryptCmd.Flags().Lookup("report"))
  RootCmd.AddCommand(decryptCmd)
}

func decryptFile(w *tred.Worker, file string, withBar bool) (string, []byte, error) {
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
  
  var r io.Reader
  r = input
  if ! viper.GetBool("encrypt.silent") && withBar {
    info, _ := input.Stat()
    bar := getProgressBar(info)
    bar.Start()
    defer bar.Finish()
    r = bar.NewProxyReader(input)
  }
  res, err := w.Decrypt(r, output)
  if err == nil {
    if viper.GetBool("decrypt.clean") {
      defer os.Remove(file)
    }
  } else {
    os.Remove(output.Name())
    return "", nil, err
  }
  return filepath.Base(output.Name()), res.Checksum, err
}

func validateEntry(index map[string]string, file, checksum string) bool {
  if len(index) > 0 {
    if _, ok := index[file]; ok {
      if index[file] != checksum {
        return false
      }
    }
  }
  return true
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
  
  // Get worker instance
  conf := tred.DefaultConfig(key)
  conf.Cipher = cs
  w, err := tred.NewWorker(conf)
  if err != nil {
    return err
  }
  fmt.Printf("\n")
  
  // Load index if provided
  index := make(map[string]string)
  if viper.GetString("decrypt.report") != "" {
    rp, err := filepath.Abs(viper.GetString("decrypt.report"))
    if err != nil {
      return err
    }
    rf, err := ioutil.ReadFile(rp)
    if err != nil {
      return err
    }
    err = json.Unmarshal(rf, &index)
    if err != nil {
      return err
    }
  }
  
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
      if ! file.IsDir() && ! strings.HasPrefix(file.Name(), ".") {
        wg.Add(1)
        go func(file os.FileInfo, report map[string]string) {
          f, checksum, err := decryptFile(w, filepath.Join(path, file.Name()), false)
          if err == nil {
            digest := fmt.Sprintf("%x", checksum)
            if ! validateEntry(index, f, digest) {
              fmt.Printf("invalid checksum value for entry: %s\n", f)
            }
            if ! viper.GetBool("decrypt.silent") {
              fmt.Printf(">> %s\n", f)
            }
            report[f] = digest
          } else {
            fmt.Printf("ERROR: %s for: %s\n", err, file.Name())
          }
          wg.Done()
        }(file, report)
      }
    }
    wg.Wait()
  } else {
    // Process single file
    file, checksum, err := decryptFile(w, path, true)
    if err != nil {
      return err
    }
    digest := fmt.Sprintf("%x", checksum)
    if ! validateEntry(index, file, digest) {
      fmt.Printf("invalid checksum value for entry: %s\n", file)
    }
    report[file] = digest
  }
  
  if ! viper.GetBool("decrypt.silent") {
    fmt.Printf("=== Done in: %v\n", time.Since(start))
  }
  return nil
}
