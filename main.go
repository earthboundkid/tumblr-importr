package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/kezhuw/toml"
	"github.com/pkg/errors"
)

const debug = true

func main() {
	if !debug {
		log.SetOutput(ioutil.Discard)
	}

	log.Println("Reading toml...")

	data, err := ioutil.ReadFile("./tumblr.toml")
	die(errors.Wrap(err, "could not read tumblr.toml"))

	var (
		blog, key string
	)

	err = toml.Unmarshal(data, &struct {
		Blog *string
		Key  *string
	}{&blog, &key})
	die(errors.Wrap(err, "could not parse config file"))

	log.Println("Starting processors")

	// Todo configure me
	// TODO what if you don't want images?
	i := NewImageProcessor()
	pp := NewPostProcessor(i)
	tc := NewTumblrClient(blog, key, pp)

	die(Countdown(tc.Wait))

	// TODO:
	// Move into sub-directory
	// Write documentation

	// Benchmark networking code
}

func die(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

func Countdown(f func() error) error {
	ec := make(chan error)
	go func() {
		ec <- f()
	}()

	// Move cursor down one before starting
	fmt.Println()
	const (
		cursorUp  = "\033[1A"
		eraseLine = "\033[K"
	)

	ticker := time.NewTicker(100 * time.Millisecond)
	start := time.Now()
	var lastBdl, lastSpeed Size

	for {
		select {
		case <-ticker.C:
			bdl := Size(atomic.LoadInt64(bytesDownloaded))
			// Average speed with last speed and total speed
			totalSpeed := Size(bdl) / Size(time.Since(start)) * Size(time.Second)
			speed := Size(bdl-lastBdl) * 10
			speed = (lastSpeed + totalSpeed + speed) / 3
			fmt.Printf("%s\rDownloaded %s @ %s/s%s\n",
				cursorUp, bdl, speed, eraseLine)
			lastBdl, lastSpeed = bdl, speed
		case err := <-ec:
			return err
		}
	}
}
