package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"go.bryk.io/pkg/crypto/tred"
	xlog "go.bryk.io/pkg/log"
)

type job struct {
	file    string
	showBar bool
	encrypt bool
	shred   bool
}

type pool struct {
	log     xlog.Logger
	sink    chan job
	count   int
	workers []*worker
	wg      sync.WaitGroup
}

type worker struct {
	name string
	jobs <-chan job
	log  xlog.Logger
	tw   *tred.Worker
	wg   *sync.WaitGroup
}

func newPool(size int, key []byte, cipher string, log xlog.Logger) (*pool, error) {
	// New pool
	p := &pool{
		wg:      sync.WaitGroup{},
		log:     log,
		sink:    make(chan job),
		workers: []*worker{},
	}

	// Setup workers
	for i := 0; i < size; i++ {
		tw, err := getWorker(key, cipher)
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("worker-%d", i+1)
		w := &worker{
			name: name,
			jobs: p.sink,
			log:  log.Sub(xlog.Fields{"component": name}),
			tw:   tw,
			wg:   &p.wg,
		}
		p.workers = append(p.workers, w)
	}

	// Start workers processing in the background and return pool instance
	for _, w := range p.workers {
		go w.run()
	}
	return p, nil
}

func (p *pool) add(j job) {
	p.count++
	p.wg.Add(1)
	p.sink <- j
}

func (p *pool) done() {
	// Block and wait for all jobs to be completed, terminate workers and return
	p.wg.Wait()
	close(p.sink)
}

func (w *worker) run() {
	for j := range w.jobs {
		// Run operation
		var err error
		if j.shred {
			err = w.shred(j.file, j.showBar)
		} else if j.encrypt {
			err = w.encrypt(j.file, j.showBar)
		} else {
			err = w.decrypt(j.file, j.showBar)
		}

		// Log result
		if err != nil {
			w.log.Error(err)
		} else {
			w.log.Info(j.file)
		}

		// Mark entry as completed
		w.wg.Done()
	}
}

// nolint: gosec
func (w *worker) encrypt(file string, withBar bool) error {
	// Open input file
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer func() {
		if err := input.Close(); err != nil {
			w.log.Error(err)
		}
	}()

	// Create new file for the ciphertext
	output, err := os.Create(fmt.Sprintf("%s%s", file, viper.GetString("encrypt.suffix")))
	if err != nil {
		return err
	}
	defer func() {
		if err := output.Close(); err != nil {
			w.log.Error(err)
		}
	}()

	// Get progress bar
	var r io.Reader
	r = input
	if !viper.GetBool("encrypt.silent") && withBar {
		bar := getProgressBar(input)
		bar.Start()
		defer bar.Finish()
		r = bar.NewProxyReader(input)
	}

	// Process input
	_, err = w.tw.Encrypt(r, output)
	if err == nil {
		if viper.GetBool("encrypt.clean") {
			if err := os.Remove(file); err != nil {
				w.log.WithField("file", file).Warning("failed to remove file")
			}
		}
	}
	return err
}

// nolint: gosec
func (w *worker) decrypt(file string, withBar bool) error {
	// Open input file
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}
	defer func() {
		if err := input.Close(); err != nil {
			w.log.Error(err)
		}
	}()

	// Get output holder
	output, err := os.Create(strings.Replace(file, viper.GetString("decrypt.suffix"), "", 1))
	if err != nil {
		return err
	}
	defer func() {
		if err := output.Close(); err != nil {
			w.log.Error(err)
		}
	}()

	// Get progress bar
	var r io.Reader
	r = input
	if !viper.GetBool("decrypt.silent") && withBar {
		bar := getProgressBar(input)
		bar.Start()
		defer bar.Finish()
		r = bar.NewProxyReader(input)
	}

	// Decrypt input
	_, err = w.tw.Decrypt(r, output)
	// nolint: nestif
	if err == nil {
		// Remove encrypted file
		if viper.GetBool("decrypt.clean") {
			if err = os.Remove(file); err != nil {
				w.log.WithField("file", file).Warning("failed to remove file")
			}
		}
	} else {
		// Remove partially decrypted file
		if err := os.Remove(output.Name()); err != nil {
			w.log.WithField("file", file).Warning("failed to remove partially decrypted file")
		}
	}
	return err
}

// nolint: gosec
func (w *worker) shred(file string, withBar bool) error {
	// Open input file
	input, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}

	// Create new file for the ciphertext
	output, err := os.Create(fmt.Sprintf("%s%s", file, viper.GetString("encrypt.suffix")))
	if err != nil {
		return err
	}

	// Get progress bar
	var r io.Reader
	r = input
	if !viper.GetBool("shred.silent") && withBar {
		bar := getProgressBar(input)
		bar.Start()
		defer bar.Finish()
		r = bar.NewProxyReader(input)
	}

	// Encrypt file in-place
	_, err = w.tw.Encrypt(r, output)
	if err == nil {
		_ = input.Close()
		_ = output.Close()
		if err := os.Remove(input.Name()); err != nil {
			w.log.WithField("file", file).Warning("failed to remove file")
		}
		if err := os.Remove(output.Name()); err != nil {
			w.log.WithField("file", file).Warning("failed to remove file")
		}
	}
	return err
}
